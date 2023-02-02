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

// Package utils provides helper methods related to ovs and ip parsing
package utils

import (
	"net"

	"github.com/pkg/errors"
)

// ParseTunnelIP parses and maps the given ip cidr with available network interface
func ParseTunnelIP(srcIP net.IP) (net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get list of network interfaces")
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
			default:
				continue
			}
		}
	}
	return nil, errors.New("error in parsing tunnel ip address")
}
