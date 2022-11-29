// Copyright (c) 2022 Nordix Foundation.
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

package mtu

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vlan"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"

	ovsutil "github.com/networkservicemesh/sdk-ovs/pkg/tools/utils"
)

const (
	viaLabel = "via"
)

type mtuClient struct {
	l2Connections map[string]*ovsutil.L2ConnectionPoint
	mtus          mtuMap
}

// NewClient - returns client chain element to manage vlan MTU
func NewClient(l2Connections map[string]*ovsutil.L2ConnectionPoint) networkservice.NetworkServiceClient {
	return &mtuClient{
		l2Connections: l2Connections,
	}
}

func (m *mtuClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	postponeCtxFunc := postpone.ContextWithValues(ctx)
	logger := log.FromContext(ctx).WithField("vlanClient", "Request")

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	if mechanism := vlan.ToMechanism(conn.GetMechanism()); mechanism != nil {
		viaSelector, ok := conn.GetLabels()[viaLabel]
		if !ok {
			return conn, nil
		}
		l2Point, ok := m.l2Connections[viaSelector]
		if !ok {
			return conn, nil
		}
		if l2Point.Interface == "" {
			return conn, nil
		}
		localMTU, loaded := m.mtus.Load(l2Point.Interface)
		if !loaded {
			localMTU, err = getMTU(l2Point, logger)
			if err != nil {
				closeCtx, cancelClose := postponeCtxFunc()
				defer cancelClose()
				if _, closeErr := m.Close(closeCtx, conn, opts...); closeErr != nil {
					err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
				}
				return nil, err
			}
			m.mtus.Store(l2Point.Interface, localMTU)
		}
		if localMTU > 0 && (conn.GetContext().GetMTU() > localMTU || conn.GetContext().GetMTU() == 0) {
			if conn.GetContext() == nil {
				conn.Context = &networkservice.ConnectionContext{}
			}
			conn.GetContext().MTU = localMTU
		}
	}
	return conn, nil
}

func (m *mtuClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
