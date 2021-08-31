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

// +build linux

// Package kernel implements client and server kernel mechanism chain element supports
// both kernel and smartvf datapath
package kernel

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk-ovs/pkg/tools/ifnames"
)

type kernelClient struct {
	bridgeName string
}

// NewClient returns a client chain element implementing kernel mechanism with veth pair or smartvf
func NewClient(bridgeName string) networkservice.NetworkServiceClient {
	return &kernelClient{bridgeName}
}

func (c *kernelClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	logger := log.FromContext(ctx).WithField("kernelClient", "Request")

	request.MechanismPreferences = append(request.MechanismPreferences, &networkservice.Mechanism{
		Cls:  cls.LOCAL,
		Type: kernel.MECHANISM,
	})

	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil || request.GetConnection().GetNextPathSegment() != nil {
		return conn, err
	}

	_, exists := conn.GetMechanism().GetParameters()[common.PCIAddressKey]
	if exists {
		if err = setupVF(ctx, logger, conn, c.bridgeName, metadata.IsClient(c)); err != nil {
			closeCtx, cancelClose := postponeCtxFunc()
			defer cancelClose()
			if _, closeErr := c.Close(closeCtx, conn, opts...); closeErr != nil {
				logger.Errorf("failed to close failed connection: %s %s", conn.GetId(), closeErr.Error())
			}
		}
	} else {
		if err = setupVeth(ctx, logger, conn, c.bridgeName, metadata.IsClient(c)); err != nil {
			closeCtx, cancelClose := postponeCtxFunc()
			defer cancelClose()
			if _, closeErr := c.Close(closeCtx, conn, opts...); closeErr != nil {
				logger.Errorf("failed to close failed connection: %s %s", conn.GetId(), closeErr.Error())
			}
		}
	}

	return conn, err
}

func (c *kernelClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	logger := log.FromContext(ctx).WithField("kernelClient", "Close")
	_, err := next.Client(ctx).Close(ctx, conn, opts...)

	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil {
		var kernelMechErr error
		ovsPortInfo, exists := ifnames.Load(ctx, metadata.IsClient(c))
		if exists {
			if !ovsPortInfo.IsVfRepresentor {
				kernelMechErr = resetVeth(ctx, logger, conn, c.bridgeName, metadata.IsClient(c))
			} else {
				kernelMechErr = resetVF(logger, ovsPortInfo, c.bridgeName)
			}
		}

		if err != nil && kernelMechErr != nil {
			return nil, errors.Wrap(err, kernelMechErr.Error())
		}
		if kernelMechErr != nil {
			return nil, kernelMechErr
		}
	}

	return &empty.Empty{}, err
}
