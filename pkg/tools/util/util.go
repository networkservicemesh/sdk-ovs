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

package util

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"
)

// Get Port number from Interface name in OVS
func GetInterfaceOfPort(logger log.Logger, interfaceName string) (int, error) {
	var portNo, count int
	count = 5
	for count > 0 {
		ofPort, stdErr, err := util.RunOVSVsctl("--if-exists", "get", "interface", interfaceName, "ofport")
		if err != nil {
			return -1, err
		}
		if stdErr != "" {
			logger.Infof("ovsutils: error occured while retrieving of port for interface %s - %s", interfaceName, stdErr)
		}
		portNo, err = strconv.Atoi(ofPort)
		if err != nil {
			return -1, err
		}
		if portNo == 0 {
			logger.Infof("ovsutils: got port number %d for interface %s, retrying", portNo, interfaceName)
			count = count - 1
			time.Sleep(500 * time.Millisecond)
			continue
		} else {
			break
		}
	}
	return portNo, nil
}

func ParseTunnelIP(srcIP net.IP) (net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				ipAddr := v.IP
				mask := v.Mask
				ipMask := ipAddr.Mask(mask)
				if v.IP.Equal(srcIP) || ipMask.Equal(srcIP) {
					return v.IP, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("error in parsing tunnel ip address: %v", srcIP)
}
