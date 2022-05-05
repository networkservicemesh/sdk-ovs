// Copyright (c) 2021-2022 Nordix Foundation.
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
	"strings"
	"sync"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-ovs/pkg/tools/ifnames"
	ovsutil "github.com/networkservicemesh/sdk-ovs/pkg/tools/utils"
)

func add(ctx context.Context, logger log.Logger, conn *networkservice.Connection, bridgeName string,
	vxlanInterfacesMutex sync.Locker, vxlanRefCountMap map[string]int, isClient bool) error {
	if mechanism := vxlan.ToMechanism(conn.GetMechanism()); mechanism != nil {
		if _, ok := ifnames.Load(ctx, isClient); ok {
			return nil
		}

		if mechanism.SrcIP() == nil {
			return errors.Errorf("no vxlan SrcIP not provided")
		}
		if mechanism.DstIP() == nil {
			return errors.Errorf("no vxlan DstIP not provided")
		}
		var egressIP, remoteIP net.IP
		if !isClient {
			egressIP = mechanism.DstIP()
			remoteIP = mechanism.SrcIP()
		} else {
			remoteIP = mechanism.DstIP()
			egressIP = mechanism.SrcIP()
		}
		ovsTunnelName := getTunnelPortName(remoteIP.String())
		vxlanInterfacesMutex.Lock()
		defer vxlanInterfacesMutex.Unlock()
		if _, exists := vxlanRefCountMap[ovsTunnelName]; !exists {
			if err := newVXLAN(bridgeName, ovsTunnelName, egressIP, remoteIP); err != nil {
				return err
			}
			vxlanRefCountMap[ovsTunnelName] = 0
		}
		vxlanRefCountMap[ovsTunnelName]++
		ovsTunnelPortNum, err := ovsutil.GetInterfaceOfPort(logger, ovsTunnelName)
		if err != nil {
			return err
		}
		ifnames.Store(ctx, isClient, &ifnames.OvsPortInfo{PortName: ovsTunnelName,
			PortNo: ovsTunnelPortNum, IsTunnelPort: true, VNI: mechanism.VNI()})
	}
	return nil
}

func getTunnelPortName(remoteIP string) string {
	return "v" + strings.ReplaceAll(remoteIP, ".", "")
}

func remove(conn *networkservice.Connection, bridgeName string, vxlanInterfacesMutex sync.Locker,
	vxlanRefCountMap map[string]int, isClient bool) error {
	if mechanism := vxlan.ToMechanism(conn.GetMechanism()); mechanism != nil {
		var remoteIP net.IP
		if !isClient {
			remoteIP = mechanism.SrcIP()
		} else {
			remoteIP = mechanism.DstIP()
		}
		ovsTunnelName := getTunnelPortName(remoteIP.String())
		vxlanInterfacesMutex.Lock()
		defer vxlanInterfacesMutex.Unlock()
		if count := vxlanRefCountMap[ovsTunnelName]; count == 1 {
			if err := deleteVXLAN(bridgeName, ovsTunnelName); err != nil {
				return errors.Wrapf(err, "failed to delete VXLAN interface")
			}
			delete(vxlanRefCountMap, ovsTunnelName)
		} else if count := vxlanRefCountMap[ovsTunnelName]; count > 1 {
			vxlanRefCountMap[ovsTunnelName]--
		}
	}
	return nil
}

// newVXLAN creates a VXLAN interface instance in OVS
func newVXLAN(bridgeName, ovsTunnelName string, egressIP, remoteIP net.IP) error {
	/* Populate the VXLAN interface configuration */
	localOptions := "options:local_ip=" + egressIP.String()
	remoteOptions := "options:remote_ip=" + remoteIP.String()
	stdout, stderr, err := util.RunOVSVsctl("--", "--may-exist", "add-port", bridgeName, ovsTunnelName,
		"--", "set", "interface", ovsTunnelName, "type=vxlan", localOptions,
		remoteOptions, "options:key=flow")
	if err != nil {
		return errors.Errorf("Failed to add port %s to %s, stdout: %q, stderr: %q,"+
			" error: %v", ovsTunnelName, bridgeName, stdout, stderr, err)
	}
	return nil
}

func deleteVXLAN(bridgeName, ovsTunnelPort string) error {
	/* Populate the VXLAN interface configuration */
	stdout, stderr, err := util.RunOVSVsctl("del-port", bridgeName, ovsTunnelPort)
	if err != nil {
		return errors.Errorf("Failed to delete port %s to %s, stdout: %q, stderr: %q,"+
			" error: %v", ovsTunnelPort, bridgeName, stdout, stderr, err)
	}
	return nil
}
