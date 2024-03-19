// Copyright (c) 2021-2024 Nordix Foundation.
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

package vxlan

import (
	"context"
	"net"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"

	"github.com/networkservicemesh/sdk-ovs/pkg/tools/ifnames"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/vxlan/vni"
)

type vxlanServer struct {
	bridgeName           string
	vxlanInterfacesMutex sync.Locker
	vxlanInterfacesMap   map[string]int
}

// NewServer - returns a new server for the vxlan remote mechanism
func NewServer(tunnelIP net.IP, bridgeName string, mutex sync.Locker, vxlanRefCountMap map[string]int, options ...Option) networkservice.NetworkServiceServer {
	opts := &vxlanOptions{
		vxlanPort: vxlanDefaultPort,
	}
	for _, opt := range options {
		opt(opts)
	}
	return chain.NewNetworkServiceServer(
		vni.NewServer(tunnelIP, vni.WithTunnelPort(opts.vxlanPort)),
		&vxlanServer{
			bridgeName: bridgeName, vxlanInterfacesMutex: mutex, vxlanInterfacesMap: vxlanRefCountMap,
		},
	)
}

func (v *vxlanServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	logger := log.FromContext(ctx).WithField("vxlanServer", "Request")

	_, isEstablished := ifnames.Load(ctx, metadata.IsClient(v))

	if !isEstablished {
		if err := add(ctx, logger, request.GetConnection(), v.bridgeName, v.vxlanInterfacesMutex, v.vxlanInterfacesMap, metadata.IsClient(v)); err != nil {
			return nil, err
		}
	}

	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil && !isEstablished {
		closeCtx, cancelClose := postponeCtxFunc()
		defer cancelClose()
		if _, exists := ifnames.LoadAndDelete(closeCtx, metadata.IsClient(v)); exists {
			if vxlanServerErr := remove(request.GetConnection(), v.bridgeName, v.vxlanInterfacesMutex, v.vxlanInterfacesMap, metadata.IsClient(v)); vxlanServerErr != nil {
				err = errors.Wrapf(err, "connection closed with error: %s", vxlanServerErr.Error())
			}
		}
		return nil, err
	}

	return conn, err
}

func (v *vxlanServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	_, err := next.Server(ctx).Close(ctx, conn)
	if mechanism := vxlan.ToMechanism(conn.GetMechanism()); mechanism != nil {
		vxlanServerErr := remove(conn, v.bridgeName, v.vxlanInterfacesMutex, v.vxlanInterfacesMap, metadata.IsClient(v))
		ifnames.Delete(ctx, metadata.IsClient(v))

		if err != nil && vxlanServerErr != nil {
			return nil, errors.Wrap(err, vxlanServerErr.Error())
		}
		if vxlanServerErr != nil {
			return nil, vxlanServerErr
		}
	}
	return &empty.Empty{}, err
}
