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

package utils

import (
	"context"
	"strconv"
	"time"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"
	kexec "k8s.io/utils/exec"
)

// GetInterfaceOfPort get Port number from Interface name in OVS
func GetInterfaceOfPort(logger log.Logger, interfaceName string) (int, error) {
	var portNo, count int
	count = 5
	for count > 0 {
		ofPort, stdErr, err := util.RunOVSVsctl("--if-exists", "get", "interface", interfaceName, "ofport")
		if err != nil {
			return -1, err
		}
		if stdErr != "" {
			logger.Infof("error occurred while retrieving of port for interface %s - %s", interfaceName, stdErr)
		}
		portNo, err = strconv.Atoi(ofPort)
		if err != nil {
			return -1, err
		}
		if portNo == 0 {
			logger.Infof("got port number %d for interface %s, retrying", portNo, interfaceName)
			count--
			time.Sleep(500 * time.Millisecond)
			continue
		} else {
			break
		}
	}
	return portNo, nil
}

// ConfigureOvS creates ovs bridge and make it as an integration bridge
func ConfigureOvS(ctx context.Context, bridgeName string) {
	// Initialize the ovs utility wrapper.
	exec := kexec.New()
	if err := util.SetExec(exec); err != nil {
		log.FromContext(ctx).Warnf("failed to initialize ovs exec helper: %v", err)
	}

	// Create ovs bridge for client and endpoint connections
	stdout, stderr, err := util.RunOVSVsctl("--", "--may-exist", "add-br", bridgeName)
	if err != nil {
		log.FromContext(ctx).Warnf("Failed to add bridge %s, stdout: %q, stderr: %q, error: %v", bridgeName, stdout, stderr, err)
	}

	// Clean the flows from the above created ovs bridge
	stdout, stderr, err = util.RunOVSOfctl("del-flows", bridgeName)
	if err != nil {
		log.FromContext(ctx).Warnf("Failed to cleanup flows on %s "+
			"stdout: %q, stderr: %q, error: %v", bridgeName, stdout, stderr, err)
	}
}
