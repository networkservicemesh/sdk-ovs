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

package kernel

import (
	"context"
	"fmt"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"

	"github.com/networkservicemesh/sdk-ovs/pkg/tools/ifnames"
	ovsutil "github.com/networkservicemesh/sdk-ovs/pkg/tools/util"
)

const (
	srcPrefix = "tapsrc"
	dstPrefix = "tapdst"
	cVETHMTU  = 16000
)

func setupVeth(ctx context.Context, logger log.Logger, conn *networkservice.Connection, bridgeName string, isClient bool) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism == nil {
		return nil
	}
	if _, ok := ifnames.Load(ctx, isClient); ok {
		return nil
	}
	contIfName := GetInterfaceName(conn, isClient)
	hostIfName := GetOvsInterfaceName(conn, isClient)
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
	ifnames.Store(ctx, isClient, ifnames.OvsPortInfo{PortName: hostIfName, PortNo: portNo, IsTunnelPort: false})
	return nil
}

func resetVeth(ctx context.Context, logger log.Logger, conn *networkservice.Connection, bridgeName string, isClient bool) error {
	ifaceName := GetOvsInterfaceName(conn, isClient)
	/* delete the port from ovs bridge */
	stdout, stderr, err := util.RunOVSVsctl("del-port", bridgeName, ifaceName)
	if err != nil {
		logger.Errorf("Failed to delete port %s from %s, stdout: %q, stderr: %q,"+
			" error: %v", ifaceName, bridgeName, stdout, stderr, err)
	}

	/* Get a link object for the interface */
	ifaceLink, err := netlink.LinkByName(ifaceName)
	if err != nil {
		return errors.Errorf("failed to get link for %q - %v", ifaceName, err)
	}

	/* Delete the VETH pair - host namespace */
	if err := netlink.LinkDel(ifaceLink); err != nil {
		return errors.Errorf("local: failed to delete the VETH pair - %v", err)
	}
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

func GetInterfaceName(conn *networkservice.Connection, isClient bool) string {
	namingConn := conn.Clone()
	namingConn.Id = namingConn.GetPrevPathSegment().GetId()
	if isClient {
		namingConn.Id = namingConn.GetNextPathSegment().GetId()
	}
	return kernel.ToMechanism(conn.GetMechanism()).GetInterfaceName(namingConn)
}

func GetOvsInterfaceName(conn *networkservice.Connection, isClient bool) string {
	namingConn := conn.Clone()
	prefix := srcPrefix
	namingConn.Id = namingConn.GetPrevPathSegment().GetId()
	if isClient {
		prefix = dstPrefix
		namingConn.Id = namingConn.GetNextPathSegment().GetId()
	}
	name := fmt.Sprintf("%s-%s", prefix, conn.GetId())
	if len(name) > kernel.LinuxIfMaxLength {
		name = name[:kernel.LinuxIfMaxLength]
	}
	return name
}
