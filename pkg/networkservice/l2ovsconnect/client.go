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
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk-ovs/pkg/tools/ifnames"
)

type l2ConnectClient struct {
	bridgeName string
}

// NewClient creates l2 connect client
func NewClient(bridgeName string) networkservice.NetworkServiceClient {
	return &l2ConnectClient{bridgeName}
}

func (c *l2ConnectClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	logger := log.FromContext(ctx).WithField("l2ConnectClient", "Request")
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil || request.GetConnection().GetNextPathSegment() != nil {
		return conn, err
	}

	if err := addDel(ctx, logger, c.bridgeName, true); err != nil {
		if _, closeErr := c.Close(ctx, conn, opts...); closeErr != nil {
			logger.Errorf("failed to close failed connection: %s %s", conn.GetId(), closeErr.Error())
		}
		return conn, err
	}
	return conn, nil
}

func (c *l2ConnectClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	logger := log.FromContext(ctx).WithField("l2ConnectClient", "Close")
	_, err := next.Client(ctx).Close(ctx, conn, opts...)

	l2ConnectErr := addDel(ctx, logger, c.bridgeName, false)
	ifnames.Delete(ctx, metadata.IsClient(c))

	if err != nil && l2ConnectErr != nil {
		return nil, errors.Wrap(err, l2ConnectErr.Error())
	}
	if l2ConnectErr != nil {
		return nil, l2ConnectErr
	}

	return &empty.Empty{}, err
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
