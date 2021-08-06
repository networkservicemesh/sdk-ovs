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

package kernel

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/resourcepool"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk-ovs/pkg/tools/ifnames"
)

type kernelServer struct {
	bridgeName string
}

// NewServer - return a new Server chain element implementing the kernel mechanism with veth pair or smartvf
func NewServer(bridgeName string) networkservice.NetworkServiceServer {
	return &kernelServer{bridgeName}
}

// NewClient create a kernel server chain element which would be useful to do network plumbing
// for service client container
func (k *kernelServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	logger := log.FromContext(ctx).WithField("kernelServer", "Request")
	isEstablished := request.GetConnection().GetNextPathSegment() != nil
	_, exists := request.GetConnection().GetMechanism().GetParameters()[resourcepool.TokenIDKey]
	if !exists && !isEstablished {
		if err := setupVeth(ctx, logger, request.Connection, k.bridgeName, metadata.IsClient(k)); err != nil {
			_ = resetVeth(logger, request.Connection, k.bridgeName, metadata.IsClient(k))
			return nil, err
		}
	}
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil && !exists && !isEstablished {
		_ = resetVeth(logger, request.Connection, k.bridgeName, metadata.IsClient(k))
		return nil, err
	}
	if exists && !isEstablished {
		if err := setupVF(ctx, logger, request.Connection, k.bridgeName, metadata.IsClient(k)); err != nil {
			return nil, err
		}
	}
	return conn, err
}

func (k *kernelServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	logger := log.FromContext(ctx).WithField("kernelServer", "Close")
	ovsPortInfo, exists := ifnames.Load(ctx, true)
	if exists {
		if !ovsPortInfo.IsVfRepresentor {
			if err := resetVeth(logger, conn, k.bridgeName, metadata.IsClient(k)); err != nil {
				return nil, err
			}
		} else {
			if err := resetVF(logger, ovsPortInfo, k.bridgeName); err != nil {
				return nil, err
			}
		}
	}

	return next.Server(ctx).Close(ctx, conn)
}
