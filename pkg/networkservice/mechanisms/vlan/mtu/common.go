// Copyright (c) 2022 Nordix Foundation.
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

package mtu

import (
	"time"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"

	ovsutil "github.com/networkservicemesh/sdk-ovs/pkg/tools/utils"
)

func getMTU(l2CP *ovsutil.L2ConnectionPoint, logger log.Logger) (uint32, error) {
	now := time.Now()
	link, err := netlink.LinkByName(l2CP.Interface)
	if err != nil {
		return 0, nil
	}
	mtu := link.Attrs().MTU
	logger.WithField("link.Name", link.Attrs().Name).
		WithField("link.MTU", mtu).
		WithField("duration", time.Since(now)).
		WithField("netlink", "LinkByName").Debug("completed")
	if mtu >= 0 && mtu <= 65535 {
		return uint32(mtu), nil
	}

	return 0, errors.New("invalid MTU value")
}
