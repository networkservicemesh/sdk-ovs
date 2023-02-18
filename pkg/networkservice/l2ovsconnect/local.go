// Copyright (c) 2021 Nordix Foundation.
//
// Copyright (c) 2023 Cisco and/or its affiliates.
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
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-ovs/pkg/tools/ifnames"
)

func createLocalCrossConnect(logger log.Logger, bridgeName string, endpointOvsPortInfo,
	clientOvsPortInfo *ifnames.OvsPortInfo) error {
	var ofRuleToClient, ofRuleToEndpoint string
	if endpointOvsPortInfo.VlanID > 0 {
		ofRuleToClient = fmt.Sprintf("priority=100,in_port=%d,dl_vlan=%d,"+
			"actions=strip_vlan,output:%d", endpointOvsPortInfo.PortNo, endpointOvsPortInfo.VlanID,
			clientOvsPortInfo.PortNo)
		ofRuleToEndpoint = fmt.Sprintf("priority=100,in_port=%d,"+
			"actions=push_vlan:0x8100,set_field:%d->vlan_vid,output:%d", clientOvsPortInfo.PortNo,
			endpointOvsPortInfo.VlanID+4096, endpointOvsPortInfo.PortNo)
	} else {
		ofRuleToClient = fmt.Sprintf("priority=100,in_port=%d,"+
			"actions=output:%d", endpointOvsPortInfo.PortNo, clientOvsPortInfo.PortNo)
		ofRuleToEndpoint = fmt.Sprintf("priority=100,in_port=%d,"+
			"actions=output:%d", clientOvsPortInfo.PortNo, endpointOvsPortInfo.PortNo)
	}
	stdout, stderr, err := util.RunOVSOfctl("add-flow", "-OOpenflow13", bridgeName, ofRuleToClient)
	if err != nil {
		logger.Infof("Failed to add flow on %s for port %s stdout: %s"+
			" stderr: %s, error: %v", bridgeName, endpointOvsPortInfo.PortName, stdout, stderr, err)
		return errors.Wrapf(err, "failed to add flow on %s for port %s stdout: %s stderr: %s", bridgeName, endpointOvsPortInfo.PortName, stdout, stderr)
	}
	if stderr != "" {
		logger.Errorf("Failed to add flow on %s for port %s stdout: %s"+
			" stderr: %s", bridgeName, endpointOvsPortInfo.PortName, stdout, stderr)
	}

	stdout, stderr, err = util.RunOVSOfctl("add-flow", "-OOpenflow13", bridgeName, ofRuleToEndpoint)
	if err != nil {
		logger.Errorf("Failed to add flow on %s for port %s stdout: %s"+
			" stderr: %s, error: %v", bridgeName, clientOvsPortInfo.PortName, stdout, stderr, err)
		return errors.Wrapf(err, "Failed to add flow on %s for port %s stdout: %s stderr: %s", bridgeName, clientOvsPortInfo.PortName, stdout, stderr)
	}

	if stderr != "" {
		logger.Errorf("Failed to add flow on %s for port %s stdout: %s"+
			" stderr: %s", bridgeName, clientOvsPortInfo.PortName, stdout, stderr)
	}

	endpointOvsPortInfo.IsCrossConnected = true
	clientOvsPortInfo.IsCrossConnected = true

	return nil
}

func deleteLocalCrossConnect(logger log.Logger, bridgeName string, endpointOvsPortInfo,
	clientOvsPortInfo *ifnames.OvsPortInfo) error {
	var matchForEndpoint string
	if endpointOvsPortInfo.VlanID > 0 {
		matchForEndpoint = fmt.Sprintf("in_port=%d,dl_vlan=%d", endpointOvsPortInfo.PortNo, endpointOvsPortInfo.VlanID)
	} else {
		matchForEndpoint = fmt.Sprintf("in_port=%d", endpointOvsPortInfo.PortNo)
	}
	stdout, stderr, err := util.RunOVSOfctl("del-flows", "-OOpenflow13", bridgeName, matchForEndpoint)
	if err != nil {
		logger.Errorf("Failed to delete flow on %s for port "+
			"%s, stdout: %q, stderr: %q, error: %v", bridgeName, endpointOvsPortInfo.PortName, stdout, stderr, err)
		return errors.Wrapf(err, "failed to delete flow on %s for port %s, stdout: %q, stderr: %q", bridgeName, endpointOvsPortInfo.PortName, stdout, stderr)
	}

	stdout, stderr, err = util.RunOVSOfctl("del-flows", "-OOpenflow13", bridgeName, fmt.Sprintf("in_port=%d", clientOvsPortInfo.PortNo))
	if err != nil {
		logger.Errorf("Failed to delete flow on %s for port "+
			"%s, stdout: %q, stderr: %q, error: %v", bridgeName, clientOvsPortInfo.PortName, stdout, stderr, err)
	}
	return nil
}
