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

func createLocalCrossConnect(logger log.Logger, bridgeName string, endpointOvsPortInfo,
	clientOvsPortInfo ifnames.OvsPortInfo) error {
	stdout, stderr, err := util.RunOVSOfctl("add-flow", bridgeName, fmt.Sprintf("priority=100, in_port=%d,"+
		" actions=output:%d", endpointOvsPortInfo.PortNo, clientOvsPortInfo.PortNo))
	if err != nil {
		logger.Infof("Failed to add flow on %s for port %s stdout: %s"+
			" stderr: %s, error: %v", bridgeName, endpointOvsPortInfo.PortName, stdout, stderr, err)
		return err
	}
	if stderr != "" {
		logger.Errorf("Failed to add flow on %s for port %s stdout: %s"+
			" stderr: %s", bridgeName, endpointOvsPortInfo.PortName, stdout, stderr)
	}

	stdout, stderr, err = util.RunOVSOfctl("add-flow", bridgeName, fmt.Sprintf("priority=100, in_port=%d,"+
		" actions=output:%d", clientOvsPortInfo.PortNo, endpointOvsPortInfo.PortNo))
	if err != nil {
		logger.Errorf("Failed to add flow on %s for port %s stdout: %s"+
			" stderr: %s, error: %v", bridgeName, clientOvsPortInfo.PortName, stdout, stderr, err)
		return err
	}

	if stderr != "" {
		logger.Errorf("Failed to add flow on %s for port %s stdout: %s"+
			" stderr: %s", bridgeName, clientOvsPortInfo.PortName, stdout, stderr)
	}
	return nil
}

func deleteLocalCrossConnect(logger log.Logger, bridgeName string, endpointOvsPortInfo,
	clientOvsPortInfo ifnames.OvsPortInfo) error {
	stdout, stderr, err := util.RunOVSOfctl("del-flows", bridgeName, fmt.Sprintf("in_port=%d",
		endpointOvsPortInfo.PortNo))
	if err != nil {
		logger.Errorf("Failed to delete flow on %s for port "+
			"%s, stdout: %q, stderr: %q, error: %v", bridgeName, endpointOvsPortInfo.PortName, stdout, stderr, err)
		return err
	}

	stdout, stderr, err = util.RunOVSOfctl("del-flows", bridgeName, fmt.Sprintf("in_port=%d", clientOvsPortInfo.PortNo))
	if err != nil {
		logger.Errorf("Failed to delete flow on %s for port "+
			"%s, stdout: %q, stderr: %q, error: %v", bridgeName, clientOvsPortInfo.PortName, stdout, stderr, err)
	}
	return nil
}
