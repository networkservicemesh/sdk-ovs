// Copyright (c) 2021-2024 Nordix Foundation.
//
// Copyright (c) 2023-2024 Cisco and/or its affiliates.
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

package l2ovsconnect

import (
	"fmt"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-ovs/pkg/tools/ifnames"
	"github.com/networkservicemesh/sdk-ovs/pkg/tools/utils"
)

func createRemoteCrossConnect(logger log.Logger, bridgeName string, endpointOvsPortInfo, clientOvsPortInfo *ifnames.OvsPortInfo) error {
	var (
		ovsLocalPortNum, ovsTunnelPortNum int
		ovsLocalPort, ovsTunnelPort       string
		vni, vlanID                       uint32
	)
	if endpointOvsPortInfo.IsTunnelPort {
		ovsLocalPortNum = clientOvsPortInfo.PortNo
		ovsLocalPort = clientOvsPortInfo.PortName
		ovsTunnelPortNum = endpointOvsPortInfo.PortNo
		ovsTunnelPort = endpointOvsPortInfo.PortName
		vni = endpointOvsPortInfo.VNI
	} else {
		ovsLocalPortNum = endpointOvsPortInfo.PortNo
		ovsLocalPort = endpointOvsPortInfo.PortName
		vlanID = endpointOvsPortInfo.VlanID
		ovsTunnelPortNum = clientOvsPortInfo.PortNo
		ovsTunnelPort = clientOvsPortInfo.PortName
		vni = clientOvsPortInfo.VNI
	}

	var ofRuleFrom, ofRuleTo string
	if vlanID > 0 {
		ofRuleFrom = fmt.Sprintf("priority=100,in_port=%d,dl_vlan=%d,actions=strip_vlan,set_field:%d->tun_id,output:%d",
			ovsLocalPortNum, vlanID, vni, ovsTunnelPortNum)
		ofRuleTo = fmt.Sprintf("priority=100,in_port=%d,tun_id=%d,actions=push_vlan:0x8100,set_field:%d->vlan_vid,output:%d",
			ovsTunnelPortNum, vni, vlanID+4096, ovsLocalPortNum)
	} else {
		ofRuleFrom = fmt.Sprintf("priority=100,in_port=%d,actions=set_field:%d->tun_id,output:%d",
			ovsLocalPortNum, vni, ovsTunnelPortNum)
		ofRuleTo = fmt.Sprintf("priority=100,in_port=%d,tun_id=%d,actions=output:%d", ovsTunnelPortNum, vni, ovsLocalPortNum)
	}
	w := &utils.OVSRunWrapper{Logger: logger}
	stdout, stderr, err := w.RunOVSOfctl("add-flow", "-OOpenflow13", bridgeName, ofRuleFrom)
	if err != nil {
		logger.Errorf("Failed to add flow on %s for port %s stdout: %s"+
			" stderr: %s, error: %v", bridgeName, ovsLocalPort, stdout, stderr, err)
		return errors.Wrapf(err, "failed to add flow on %s for port %s stdout: %s stderr: %s", bridgeName, ovsLocalPort, stdout, stderr)
	}
	if stderr != "" {
		logger.Errorf("Failed to add flow on %s for port %s stdout: %s"+
			" stderr: %s", bridgeName, ovsLocalPort, stdout, stderr)
	}
	stdout, stderr, err = w.RunOVSOfctl("add-flow", "-OOpenflow13", bridgeName, ofRuleTo)
	if err != nil {
		logger.Errorf("Failed to add tunnel flow on %s for port %s stdout: %s"+
			" stderr: %s, error: %v", bridgeName, ovsTunnelPort, stdout, stderr, err)
		return errors.Wrapf(err, "failed to add tunnel flow on %s for port %s stdout: %s, stderr: %s", bridgeName, ovsTunnelPort, stdout, stderr)
	}
	if stderr != "" {
		logger.Errorf("Failed to add tunnel flow on %s for port %s stdout: %s"+
			" stderr: %s", bridgeName, ovsTunnelPort, stdout, stderr)
	}

	endpointOvsPortInfo.IsCrossConnected = true
	clientOvsPortInfo.IsCrossConnected = true

	return nil
}

func deleteRemoteCrossConnect(logger log.Logger, bridgeName string, endpointOvsPortInfo, clientOvsPortInfo *ifnames.OvsPortInfo) error {
	var (
		ovsLocalPortNum, ovsTunnelPortNum int
		ovsLocalPort, ovsTunnelPort       string
		vni, vlanID                       uint32
	)
	if endpointOvsPortInfo.IsTunnelPort {
		ovsLocalPortNum = clientOvsPortInfo.PortNo
		ovsLocalPort = clientOvsPortInfo.PortName
		ovsTunnelPortNum = endpointOvsPortInfo.PortNo
		ovsTunnelPort = endpointOvsPortInfo.PortName
		vni = endpointOvsPortInfo.VNI
	} else {
		ovsLocalPortNum = endpointOvsPortInfo.PortNo
		ovsLocalPort = endpointOvsPortInfo.PortName
		vlanID = endpointOvsPortInfo.VlanID
		ovsTunnelPortNum = clientOvsPortInfo.PortNo
		ovsTunnelPort = clientOvsPortInfo.PortName
		vni = clientOvsPortInfo.VNI
	}
	var ofMatch string
	w := &utils.OVSRunWrapper{Logger: logger}
	if vlanID > 0 {
		ofMatch = fmt.Sprintf("in_port=%d,dl_vlan=%d", ovsLocalPortNum, vlanID)
	} else {
		ofMatch = fmt.Sprintf("in_port=%d", ovsLocalPortNum)
	}
	stdout, stderr, err := w.RunOVSOfctl("del-flows", "-OOpenflow13", bridgeName, ofMatch)
	if err != nil {
		logger.Errorf("Failed to delete flow on %s for port "+
			"%s, stdout: %q, stderr: %q, error: %v", bridgeName, ovsLocalPort, stdout, stderr, err)
		return errors.Wrapf(err, "Failed to delete flow on %s for port %s, stdout: %q, stderr: %q", bridgeName, ovsLocalPort, stdout, stderr)
	}
	stdout, stderr, err = w.RunOVSOfctl("del-flows", "-OOpenflow13", bridgeName, fmt.Sprintf("in_port=%d,tun_id=%d", ovsTunnelPortNum, vni))
	if err != nil {
		logger.Errorf("Failed to delete flow on %s for port "+
			"%s on VNI %d, stdout: %q, stderr: %q, error: %v", bridgeName, ovsTunnelPort, vni, stdout, stderr, err)
		return errors.Wrapf(err, "failed to delete flow on %s for port %s on VNI %d, stdout: %q, stderr: %q", bridgeName, ovsTunnelPort, vni, stdout, stderr)
	}

	return nil
}
