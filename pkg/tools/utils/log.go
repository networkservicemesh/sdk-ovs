// Copyright (c) 2024 Nordix Foundation.
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

// Package utils provides helper methods related to ovs and ip parsing
package utils

import (
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"
)

// OVSRunWrapper -
type OVSRunWrapper struct {
	Logger log.Logger
}

// RunOVSVsctl - wrapper function for github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util.RunOVSVsctl
func (w *OVSRunWrapper) RunOVSVsctl(args ...string) (stdout, stderr string, err error) {
	stdout, stderr, err = util.RunOVSVsctl(args...)
	defer func() {
		cmdStr := []string{"ovs-vsctl"}
		cmdStr = append(cmdStr, args...)
		w.Logger.WithField("RunOVSVsctl", cmdStr).
			WithField("stdout", stdout).
			Debug("completed")
	}()
	return stdout, stderr, err
}

// RunOVSOfctl - wrapper function for github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util.RunOVSOfctl
func (w *OVSRunWrapper) RunOVSOfctl(args ...string) (stdout, stderr string, err error) {
	stdout, stderr, err = util.RunOVSOfctl(args...)
	defer func() {
		cmdStr := []string{"ovs-ofctl"}
		cmdStr = append(cmdStr, args...)
		w.Logger.WithField("RunOVSOfctl", cmdStr).
			WithField("stdout", stdout).
			Debug("completed")
	}()
	return stdout, stderr, err
}
