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
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-ovs/pkg/tools/ifnames"
)

type kernelSmartVFServer struct {
	bridgeName          string
	parentIfmutex       sync.Locker
	parentIfRefCountMap map[string]int
}

// NewSmartVFServer - return a new Smart VF Server chain element for kernel mechanism
func NewSmartVFServer(bridgeName string, mutex sync.Locker, parentIfRefCountMap map[string]int) networkservice.NetworkServiceServer {
	return &kernelSmartVFServer{bridgeName: bridgeName, parentIfmutex: mutex, parentIfRefCountMap: parentIfRefCountMap}
}

// NewClient create a kernel Smart VF server chain element which would be useful to do network plumbing
// for service client container
func (k *kernelSmartVFServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	logger := log.FromContext(ctx).WithField("kernelSmartVFServer", "Request")

	_, isEstablished := ifnames.Load(ctx, metadata.IsClient(k))

	if !isEstablished {
		k.parentIfmutex.Lock()
		if vfErr := setupVF(ctx, logger, request.GetConnection(), k.bridgeName, k.parentIfRefCountMap, metadata.IsClient(k)); vfErr != nil {
			k.parentIfmutex.Unlock()
			return nil, vfErr
		}
		k.parentIfmutex.Unlock()
	}

	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil && !isEstablished {
		closeCtx, cancelClose := postponeCtxFunc()
		defer cancelClose()
		if ovsPortInfo, exists := ifnames.LoadAndDelete(closeCtx, metadata.IsClient(k)); exists {
			k.parentIfmutex.Lock()
			if kernelServerErr := resetVF(logger, ovsPortInfo, k.parentIfRefCountMap, k.bridgeName); kernelServerErr != nil {
				err = errors.Wrapf(err, "connection closed with error: %s", kernelServerErr.Error())
			}
			k.parentIfmutex.Unlock()
		}
		return nil, err
	}

	return conn, err
}

func (k *kernelSmartVFServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	logger := log.FromContext(ctx).WithField("kernelSmartVFServer", "Close")
	_, err := next.Server(ctx).Close(ctx, conn)

	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil {
		k.parentIfmutex.Lock()
		defer k.parentIfmutex.Unlock()
		var kernelServerErr error
		ovsPortInfo, exists := ifnames.LoadAndDelete(ctx, metadata.IsClient(k))
		if exists {
			kernelServerErr = resetVF(logger, ovsPortInfo, k.parentIfRefCountMap, k.bridgeName)
		}

		if err != nil && kernelServerErr != nil {
			return nil, errors.Wrap(err, kernelServerErr.Error())
		}
		if kernelServerErr != nil {
			return nil, kernelServerErr
		}
	}

	return &empty.Empty{}, err
}
