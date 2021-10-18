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
	"errors"

	"github.com/Mellanox/sriovnet"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/vfconfig"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"

	"github.com/networkservicemesh/sdk-ovs/pkg/tools/ifnames"
	ovsutil "github.com/networkservicemesh/sdk-ovs/pkg/tools/utils"
)

func setupVF(ctx context.Context, logger log.Logger, conn *networkservice.Connection, bridgeName string,
	parentIfRefCount map[string]int, isClient bool) error {
	var mechanism *kernel.Mechanism
	if mechanism = kernel.ToMechanism(conn.GetMechanism()); mechanism == nil {
		return nil
	}
	if _, ok := ifnames.Load(ctx, isClient); ok {
		return nil
	}

	vfConfig, exists := vfconfig.Load(ctx, isClient)
	if !exists {
		return errors.New("vfconfig not found")
	}
	// get smart VF representor interface. This is a host net device which represents
	// smart VF attached inside the container by device plugin. It can be considered
	// as one end of veth pair whereas other end is smartVF. The VF representor would
	// get added into ovs bridge for the control plane configuration.
	vfRepresentor, err := sriovnet.GetVfRepresentor(vfConfig.PFInterfaceName, vfConfig.VFNum)
	if err != nil {
		return err
	}
	if _, exists := parentIfRefCount[vfRepresentor]; !exists {
		stdout, stderr, err1 := util.RunOVSVsctl("--", "--may-exist", "add-port", bridgeName, vfRepresentor)
		if err1 != nil {
			logger.Errorf("Failed to add representor port %s to %s, stdout: %q, stderr: %q,"+
				" error: %v", vfRepresentor, bridgeName, stdout, stderr, err1)
			return err1
		}
		parentIfRefCount[vfRepresentor] = 0
	}
	parentIfRefCount[vfRepresentor]++
	portNo, err := ovsutil.GetInterfaceOfPort(logger, vfRepresentor)
	if err != nil {
		logger.Errorf("Failed to get OVS port number for %s interface,"+
			" error: %v", vfRepresentor, err)
		return err
	}

	ifnames.Store(ctx, isClient, &ifnames.OvsPortInfo{PortName: vfRepresentor, PortNo: portNo,
		VlanID: mechanism.GetVLAN(), IsVfRepresentor: true})
	return nil
}

func resetVF(logger log.Logger, portInfo *ifnames.OvsPortInfo, parentIfRefCountMap map[string]int, bridgeName string) error {
	/* delete the port from ovs bridge */
	var refCount int
	if count, exists := parentIfRefCountMap[portInfo.PortName]; exists {
		if count > 0 {
			refCount = count - 1
			parentIfRefCountMap[portInfo.PortName] = refCount
		}
	}
	if refCount == 0 {
		stdout, stderr, err := util.RunOVSVsctl("del-port", bridgeName, portInfo.PortName)
		if err != nil {
			logger.Errorf("Failed to delete port %s from %s, stdout: %q, stderr: %q,"+
				" error: %v", portInfo.PortName, bridgeName, stdout, stderr, err)
			return err
		}
		delete(parentIfRefCountMap, portInfo.PortName)
	}

	return nil
}
