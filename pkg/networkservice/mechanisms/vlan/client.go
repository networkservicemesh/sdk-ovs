// Copyright (c) 2021-2022 Nordix Foundation.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux
// +build linux

// Package vlan client chain element implementing remote vlan breakout mechanism
package vlan

import (
	"context"
	"fmt"

	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	vlanmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vlan"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk-ovs/pkg/networkservice/mechanisms/vlan/mtu"
	"github.com/networkservicemesh/sdk-ovs/pkg/tools/ifnames"
	ovsutil "github.com/networkservicemesh/sdk-ovs/pkg/tools/utils"
)

const (
	viaLabel = "via"
)

type vlanClient struct {
	bridgeName    string
	l2Connections map[string]*ovsutil.L2ConnectionPoint
}

// NewClient returns a client chain element implementing VLAN breakout for NS client
func NewClient(bridgeName string, l2Connections map[string]*ovsutil.L2ConnectionPoint) networkservice.NetworkServiceClient {
	return chain.NewNetworkServiceClient(
		mtu.NewClient(l2Connections),
		&vlanClient{bridgeName: bridgeName, l2Connections: l2Connections},
	)
}

func (c *vlanClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	logger := log.FromContext(ctx).WithField("vlanClient", "Request")
	mechanism := &networkservice.Mechanism{
		Cls:        cls.REMOTE,
		Type:       vlanmech.MECHANISM,
		Parameters: make(map[string]string),
	}
	request.MechanismPreferences = append(request.MechanismPreferences, mechanism)

	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	if err := c.addDelVlan(ctx, logger, conn, true); err != nil {
		closeCtx, cancelClose := postponeCtxFunc()
		defer cancelClose()
		if _, closeErr := c.Close(closeCtx, conn, opts...); closeErr != nil {
			err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
		}
		return nil, err
	}

	return conn, nil
}

func (c *vlanClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	logger := log.FromContext(ctx).WithField("vlanClient", "Close")
	_, err := next.Client(ctx).Close(ctx, conn, opts...)
	vlanMechErr := c.addDelVlan(ctx, logger, conn, false)
	if err != nil && vlanMechErr != nil {
		return nil, errors.Wrap(err, vlanMechErr.Error())
	}
	if vlanMechErr != nil {
		return nil, vlanMechErr
	}
	return &empty.Empty{}, err
}

func (c *vlanClient) addDelVlan(ctx context.Context, logger log.Logger, conn *networkservice.Connection, isAdd bool) error {
	mechanism := vlanmech.ToMechanism(conn.GetMechanism())
	if mechanism == nil {
		return nil
	}
	nsClientOvsPortInfo, ok := ifnames.Load(ctx, false)
	if !ok || (isAdd && nsClientOvsPortInfo.IsCrossConnected) {
		return nil
	}
	viaSelector, ok := conn.GetLabels()[viaLabel]
	if !ok {
		return nil
	}
	l2Point, ok := c.l2Connections[viaSelector]
	if !ok {
		return nil
	}
	if isAdd {
		// delete the ns client port from br-nsm bridge and add it into l2 connect bridge with vlan tag.
		stdout, stderr, err := util.RunOVSVsctl("del-port", c.bridgeName, nsClientOvsPortInfo.PortName)
		if err != nil {
			logger.Errorf("Failed to delete port %s from %s, stdout: %q, stderr: %q,"+
				" error: %v", nsClientOvsPortInfo.PortName, c.bridgeName, stdout, stderr, err)
			return err
		}
		stdout, stderr, err = util.RunOVSVsctl("--", "--may-exist", "add-port", l2Point.Bridge,
			nsClientOvsPortInfo.PortName, fmt.Sprintf("tag=%d", mechanism.GetVlanID()))
		if err != nil {
			logger.Errorf("Failed to add port %s to %s, stdout: %q, stderr: %q,"+
				" error: %v", nsClientOvsPortInfo.PortName, l2Point.Bridge, stdout, stderr, err)
			return err
		}
		nsClientOvsPortInfo.IsL2Connect = true
		nsClientOvsPortInfo.IsCrossConnected = true
	} else {
		stdout, stderr, err := util.RunOVSVsctl("del-port", l2Point.Bridge, nsClientOvsPortInfo.PortName)
		if err != nil {
			logger.Errorf("Failed to delete port %s from %s, stdout: %q, stderr: %q,"+
				" error: %v", nsClientOvsPortInfo.PortName, l2Point.Bridge, stdout, stderr, err)
		}
	}
	return nil
}
