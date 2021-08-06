// Copyright (c) 2021 Nordix Foundation.
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

// Package vxlan implements vxlan remote mechanism client and server chain element
package vxlan

import (
	"context"
	"net"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/vxlan/vni"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type vxlanClient struct {
	bridgeName           string
	vxlanInterfacesMutex sync.Locker
	vxlanInterfacesMap   map[string]int
}

// NewClient returns a Vxlan client chain element
func NewClient(tunnelIP net.IP, bridgeName string, mutex sync.Locker, vxlanRefCountMap map[string]int) networkservice.NetworkServiceClient {
	return chain.NewNetworkServiceClient(
		&vxlanClient{
			bridgeName: bridgeName, vxlanInterfacesMutex: mutex, vxlanInterfacesMap: vxlanRefCountMap,
		},
		vni.NewClient(tunnelIP),
	)
}

func (c *vxlanClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	logger := log.FromContext(ctx).WithField("vxlanClient", "Request")
	request.MechanismPreferences = append(request.MechanismPreferences, &networkservice.Mechanism{
		Cls:  cls.REMOTE,
		Type: vxlan.MECHANISM,
	})
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil || request.GetConnection().GetNextPathSegment() != nil {
		return conn, err
	}
	if err = add(ctx, logger, conn, c.bridgeName, c.vxlanInterfacesMutex, c.vxlanInterfacesMap, true); err != nil {
		_ = remove(ctx, conn, c.bridgeName, c.vxlanInterfacesMutex, c.vxlanInterfacesMap, true)
		if _, closeErr := next.Client(ctx).Close(ctx, conn, opts...); closeErr != nil {
			logger.Errorf("failed to close failed connection: %s %s", conn.GetId(), closeErr.Error())
		}
	}
	return conn, err
}

func (c *vxlanClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	_, err := next.Client(ctx).Close(ctx, conn, opts...)

	vxlanClientErr := remove(ctx, conn, c.bridgeName, c.vxlanInterfacesMutex, c.vxlanInterfacesMap, true)

	if err != nil && vxlanClientErr != nil {
		return nil, errors.Wrap(err, vxlanClientErr.Error())
	}
	if vxlanClientErr != nil {
		return nil, vxlanClientErr
	}

	return &empty.Empty{}, err
}
