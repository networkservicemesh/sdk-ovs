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

// Package kernel implements client and server kernel mechanism chain element supports
// both kernel and smartvf datapath
package kernel

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/resourcepool"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
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
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}
	_, exists := conn.GetMechanism().GetParameters()[resourcepool.TokenIDKey]
	if exists {
		if err := setupVF(ctx, logger, c.bridgeName, metadata.IsClient(c)); err != nil {
			return nil, err
		}
	} else {
		if err := setupVeth(ctx, logger, conn, c.bridgeName, metadata.IsClient(c)); err != nil {
			_, _ = c.Close(ctx, conn, opts...)
			return nil, err
		}
	}
	return conn, nil
}

func (c *kernelClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	logger := log.FromContext(ctx).WithField("kernelClient", "Close")
	rv, err := next.Client(ctx).Close(ctx, conn, opts...)
	if err != nil {
		return nil, err
	}
	ovsPortInfo, exists := ifnames.Load(ctx, true)
	if exists {
		if !ovsPortInfo.IsVfRepresentor {
			if err := resetVeth(logger, conn, c.bridgeName, metadata.IsClient(c)); err != nil {
				return nil, err
			}
		} else {
			if err := resetVF(logger, ovsPortInfo, c.bridgeName); err != nil {
				return nil, err
			}
		}
	}
	return rv, nil
}
