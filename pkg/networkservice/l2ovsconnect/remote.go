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

package l2ovsconnect

import (
	"fmt"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"

	"github.com/networkservicemesh/sdk-ovs/pkg/tools/ifnames"
)

func createRemoteCrossConnect(logger log.Logger, bridgeName string, endpointOvsPortInfo, clientOvsPortInfo *ifnames.OvsPortInfo) error {
	var (
		ovsLocalPortNum, ovsTunnelPortNum int
		ovsLocalPort, ovsTunnelPort       string
		vni                               uint32
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
		ovsTunnelPortNum = clientOvsPortInfo.PortNo
		ovsTunnelPort = clientOvsPortInfo.PortName
		vni = clientOvsPortInfo.VNI
	}

	stdout, stderr, err := util.RunOVSOfctl("add-flow", bridgeName,
		fmt.Sprintf("priority=100, in_port=%d, actions=set_field:%d->tun_id,output:%d",
			ovsLocalPortNum, vni, ovsTunnelPortNum))
	if err != nil {
		logger.Errorf("Failed to add flow on %s for port %s stdout: %s"+
			" stderr: %s, error: %v", bridgeName, ovsLocalPort, stdout, stderr, err)
		return err
	}
	if stderr != "" {
		logger.Errorf("Failed to add flow on %s for port %s stdout: %s"+
			" stderr: %s", bridgeName, ovsLocalPort, stdout, stderr)
	}

	stdout, stderr, err = util.RunOVSOfctl("add-flow", bridgeName, fmt.Sprintf("priority=100, in_port=%d, "+
		"tun_id=%d,actions=output:%d", ovsTunnelPortNum, vni, ovsLocalPortNum))
	if err != nil {
		logger.Errorf("Failed to add tunnel flow on %s for port %s stdout: %s"+
			" stderr: %s, error: %v", bridgeName, ovsTunnelPort, stdout, stderr, err)
		return err
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
		vni                               uint32
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
		ovsTunnelPortNum = clientOvsPortInfo.PortNo
		ovsTunnelPort = clientOvsPortInfo.PortName
		vni = clientOvsPortInfo.VNI
	}
	stdout, stderr, err := util.RunOVSOfctl("del-flows", bridgeName, fmt.Sprintf("in_port=%d", ovsLocalPortNum))
	if err != nil {
		logger.Errorf("Failed to delete flow on %s for port "+
			"%s, stdout: %q, stderr: %q, error: %v", bridgeName, ovsLocalPort, stdout, stderr, err)
		return err
	}

	stdout, stderr, err = util.RunOVSOfctl("del-flows", bridgeName, fmt.Sprintf("in_port=%d,tun_id=%d", ovsTunnelPortNum, vni))
	if err != nil {
		logger.Errorf("Failed to delete flow on %s for port "+
			"%s on VNI %d, stdout: %q, stderr: %q, error: %v", bridgeName, ovsTunnelPort, vni, stdout, stderr, err)
		return err
	}

	return nil
}
