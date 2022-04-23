// Copyright (c) 2022 Cisco and/or its affiliates.
//
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

//go:build linux
// +build linux

package utils

import (
	"context"
	"strconv"
	"time"

	"github.com/vishvananda/netlink"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"
	kexec "k8s.io/utils/exec"
)

// L2ConnectionPoint contains egress point config used by clients for VLAN breakout
type L2ConnectionPoint struct {
	Interface string
	Bridge    string
}

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
func ConfigureOvS(ctx context.Context, l2Connections map[string]*L2ConnectionPoint, bridgeName string) error {
	// Initialize the ovs utility wrapper.
	exec := kexec.New()
	if err := util.SetExec(exec); err != nil {
		log.FromContext(ctx).Warnf("failed to initialize ovs exec helper: %v", err)
	}

	for _, cp := range l2Connections {
		if cp.Bridge != "" {
			// Create ovs bridge for l2 egress point
			stdout, stderr, err := util.RunOVSVsctl("--", "--may-exist", "add-br", cp.Bridge)
			if err != nil {
				log.FromContext(ctx).Warnf("Failed to add bridge %s, stdout: %q, stderr: %q, error: %v", bridgeName, stdout, stderr, err)
			}
		}
		if cp.Interface == "" {
			continue
		}
		err := configureL2Interface(ctx, cp)
		if err != nil {
			return err
		}
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

	return nil
}

func configureL2Interface(ctx context.Context, cp *L2ConnectionPoint) error {
	link, err := netlink.LinkByName(cp.Interface)
	if err != nil {
		return err
	}
	// TODO: find a way to flush the ip's (if exists) in one go.
	v4addr, err := netlink.AddrList(link, netlink.FAMILY_V4)
	if err != nil {
		return err
	}
	for idx := range v4addr {
		err = netlink.AddrDel(link, &v4addr[idx])
		if err != nil {
			return err
		}
	}
	v6addr, err := netlink.AddrList(link, netlink.FAMILY_V6)
	if err != nil {
		return err
	}
	for idx := range v6addr {
		err = netlink.AddrDel(link, &v6addr[idx])
		if err != nil {
			return err
		}
	}
	stdout, stderr, err := util.RunOVSVsctl("--", "--may-exist", "add-port", cp.Bridge, cp.Interface)
	if err != nil {
		log.FromContext(ctx).Errorf("Failed to add l2 egress port %s to %s, stdout: %q, stderr: %q,"+
			" error: %v", cp.Interface, cp.Bridge, stdout, stderr, err)
		return err
	}
	link, err = netlink.LinkByName(cp.Bridge)
	if err != nil {
		return err
	}
	for idx := range v4addr {
		err = netlink.AddrAdd(link, &v4addr[idx])
		if err != nil {
			return err
		}
	}
	for idx := range v6addr {
		err = netlink.AddrAdd(link, &v6addr[idx])
		if err != nil {
			return err
		}
	}
	return nil
}
