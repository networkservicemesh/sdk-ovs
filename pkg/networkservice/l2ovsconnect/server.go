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

// Package l2ovsconnect chain element which cross connects both client and endpoint.
// This suppports both local and remote (vxlan) cross connections.
package l2ovsconnect

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk-ovs/pkg/tools/ifnames"
)

type l2ConnectServer struct {
	bridgeName string
}

// NewServer creates l2 connect server
func NewServer(bridgeName string) networkservice.NetworkServiceServer {
	return &l2ConnectServer{bridgeName}
}

func (v *l2ConnectServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	logger := log.FromContext(ctx).WithField("l2ConnectServer", "Request")
	if err := addDel(ctx, logger, v.bridgeName, true); err != nil {
		return nil, err
	}
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		_ = addDel(ctx, logger, v.bridgeName, false)
		return nil, err
	}
	return conn, nil
}

func (v *l2ConnectServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	logger := log.FromContext(ctx).WithField("l2ConnectServer", "Close")
	if err := addDel(ctx, logger, v.bridgeName, false); err != nil {
		return nil, err
	}
	return next.Server(ctx).Close(ctx, conn)
}

func addDel(ctx context.Context, logger log.Logger, bridgeName string, addDel bool) error {
	endpointOvsPortInfo, ok := ifnames.Load(ctx, true)
	if !ok {
		return nil
	}
	clientOvsPortInfo, ok := ifnames.Load(ctx, false)
	if !ok {
		return nil
	}
	if !endpointOvsPortInfo.IsTunnelPort && !clientOvsPortInfo.IsTunnelPort {
		if addDel {
			return createLocalCrossConnect(logger, bridgeName, endpointOvsPortInfo, clientOvsPortInfo)
		}
		return deleteLocalCrossConnect(logger, bridgeName, endpointOvsPortInfo, clientOvsPortInfo)
	}
	if addDel {
		return createRemoteCrossConnect(logger, bridgeName, endpointOvsPortInfo, clientOvsPortInfo)
	}
	return deleteRemoteCrossConnect(logger, bridgeName, endpointOvsPortInfo, clientOvsPortInfo)
}
