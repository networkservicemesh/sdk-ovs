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
	"fmt"
	"strings"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/vfconfig"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"

	"github.com/networkservicemesh/sdk-ovs/pkg/tools/ifnames"
	ovsutil "github.com/networkservicemesh/sdk-ovs/pkg/tools/utils"
)

const (
	ovsPortSrcPrefix  = "tapsrc"
	ovsPortDstPrefix  = "tapdst"
	contPortSrcPrefix = "contsrc"
	contPortDstPrefix = "contdst"
	cVETHMTU          = 16000
)

func setupVeth(ctx context.Context, logger log.Logger, conn *networkservice.Connection, bridgeName string, isClient bool) error {
	var mechanism *kernel.Mechanism
	if mechanism = kernel.ToMechanism(conn.GetMechanism()); mechanism == nil {
		return nil
	}
	if _, ok := ifnames.Load(ctx, isClient); ok {
		return nil
	}

	// use intermediate contIfName to avoid interface name collision with parallel service requests from other clients.
	var hostIfName, contIfName string
	if isClient {
		hostIfName = GetInterfaceName(conn, ovsPortDstPrefix, true)
		contIfName = GetInterfaceName(conn, contPortDstPrefix, true)
	} else {
		hostIfName = GetInterfaceName(conn, ovsPortSrcPrefix, false)
		contIfName = GetInterfaceName(conn, contPortSrcPrefix, false)
	}

	if err := createInterfaces(contIfName, hostIfName); err != nil {
		return err
	}
	if err := SetInterfacesUp(logger, contIfName, hostIfName); err != nil {
		return err
	}
	stdout, stderr, err := util.RunOVSVsctl("--", "--may-exist", "add-port", bridgeName, hostIfName)
	if err != nil {
		logger.Errorf("Failed to add port %s to %s, stdout: %q, stderr: %q,"+
			" error: %v", hostIfName, bridgeName, stdout, stderr, err)
		return err
	}
	portNo, err := ovsutil.GetInterfaceOfPort(logger, hostIfName)
	if err != nil {
		logger.Errorf("Failed to get OVS port number for %s interface,"+
			" error: %v", hostIfName, err)
		return err
	}

	vfconfig.Store(ctx, isClient, &vfconfig.VFConfig{VFInterfaceName: contIfName})
	ifnames.Store(ctx, isClient, &ifnames.OvsPortInfo{PortName: hostIfName, PortNo: portNo, IsTunnelPort: false})

	return nil
}

func resetVeth(ctx context.Context, logger log.Logger, conn *networkservice.Connection, bridgeName string, isClient bool) error {
	var ifaceName string
	if isClient {
		ifaceName = GetInterfaceName(conn, ovsPortDstPrefix, true)
	} else {
		ifaceName = GetInterfaceName(conn, ovsPortSrcPrefix, false)
	}
	/* delete the port from ovs bridge */
	stdout, stderr, err := util.RunOVSVsctl("del-port", bridgeName, ifaceName)
	if err != nil {
		logger.Errorf("Failed to delete port %s from %s, stdout: %q, stderr: %q,"+
			" error: %v", ifaceName, bridgeName, stdout, stderr, err)
	}

	/* Get a link object for the interface */
	ifaceLink, err := netlink.LinkByName(ifaceName)
	if err != nil {
		if strings.Contains(err.Error(), "Link not found") {
			// link is aleady deleted
			return nil
		}
		return errors.Errorf("failed to get link for %q - %v", ifaceName, err)
	}

	/* Delete the VETH pair - host namespace */
	if err := netlink.LinkDel(ifaceLink); err != nil {
		return errors.Errorf("local: failed to delete the VETH pair - %v", err)
	}
	vfconfig.Delete(ctx, isClient)
	return nil
}

func createInterfaces(ifName, ovSPortName string) error {
	/* Create the VETH pair - host namespace */
	if err := netlink.LinkAdd(newVETH(ifName, ovSPortName)); err != nil {
		return errors.Errorf("failed to create VETH pair - %v", err)
	}
	return nil
}

// SetInterfacesUp - make the interfaces state to up
func SetInterfacesUp(logger log.Logger, ifaceNames ...string) error {
	for _, ifaceName := range ifaceNames {
		/* Get a link for the interface name */
		link, err := netlink.LinkByName(ifaceName)
		if err != nil {
			logger.Errorf("local: failed to lookup %q, %v", ifaceName, err)
			return err
		}
		/* Bring the interface Up */
		if err = netlink.LinkSetUp(link); err != nil {
			logger.Errorf("local: failed to bring %q up: %v", ifaceName, err)
			return err
		}
	}
	return nil
}

func newVETH(srcName, dstName string) *netlink.Veth {
	/* Populate the VETH interface configuration */
	return &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: srcName,
			MTU:  cVETHMTU,
		},
		PeerName: dstName,
	}
}

// GetInterfaceName get appropriate interface name for the given connection.
func GetInterfaceName(conn *networkservice.Connection, ifPrefix string, isClient bool) string {
	namingConn := conn.Clone()
	namingConn.Id = namingConn.GetPrevPathSegment().GetId()
	if isClient {
		namingConn.Id = namingConn.GetNextPathSegment().GetId()
	}
	name := fmt.Sprintf("%s-%s", ifPrefix, conn.GetId())
	if len(name) > kernel.LinuxIfMaxLength {
		name = name[:kernel.LinuxIfMaxLength]
	}
	return name
}
