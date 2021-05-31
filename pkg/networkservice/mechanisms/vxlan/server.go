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

package vxlan

import (
	"context"
	"net"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/vxlan/vni"
)

type vxlanServer struct {
	bridgeName           string
	vxlanInterfacesMutex sync.Locker
	vxlanInterfacesMap   map[string]int
}

// NewServer - returns a new server for the vxlan remote mechanism
func NewServer(tunnelIP net.IP, bridgeName string, mutex sync.Locker, vxlanRefCountMap map[string]int) networkservice.NetworkServiceServer {
	return chain.NewNetworkServiceServer(
		vni.NewServer(tunnelIP),
		&vxlanServer{
			bridgeName: bridgeName, vxlanInterfacesMutex: mutex, vxlanInterfacesMap: vxlanRefCountMap,
		},
	)
}

func (v *vxlanServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	logger := log.FromContext(ctx).WithField("vxlanServer", "Request")
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	if err := add(ctx, logger, conn, v.bridgeName, v.vxlanInterfacesMutex, v.vxlanInterfacesMap, false); err != nil {
		_, _ = v.Close(ctx, conn)
		return nil, err
	}
	return conn, nil
}

func (v *vxlanServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if err := remove(conn, v.bridgeName, v.vxlanInterfacesMutex, v.vxlanInterfacesMap, false); err != nil {
		return nil, err
	}
	return next.Server(ctx).Close(ctx, conn)
}
