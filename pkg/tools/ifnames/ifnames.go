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

// Package ifnames provides caching facilty for ovs port info using
// metadata store. This is mainly used by mechanism and l2ovsconnect
// chain elements
package ifnames

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type key struct{}

// OvsPortInfo ovs port info container
type OvsPortInfo struct {
	PortName        string
	PortNo          int
	IsTunnelPort    bool
	IsVfRepresentor bool
	VNI             uint32
}

// Store stores ovsPortInfo for the given cross connect, isClient identfies which connection it is.
func Store(ctx context.Context, isClient bool, ovsPortInfo OvsPortInfo) {
	metadata.Map(ctx, isClient).Store(key{}, ovsPortInfo)
}

// Delete deletes a specific ovsPortInfo from cache
func Delete(ctx context.Context, isClient bool) {
	metadata.Map(ctx, isClient).Delete(key{})
}

// Load retrieves ovsPortInfo from cache for a given connection
func Load(ctx context.Context, isClient bool) (value OvsPortInfo, ok bool) {
	rawValue, ok := metadata.Map(ctx, isClient).Load(key{})
	if !ok {
		return
	}
	value, ok = rawValue.(OvsPortInfo)
	return value, ok
}

// LoadOrStore retrievs ovsPortInfo from cache. If it doesn't exist, store it with given ovsPortInfo
func LoadOrStore(ctx context.Context, isClient bool, ovsPortInfo OvsPortInfo) (value OvsPortInfo, ok bool) {
	rawValue, ok := metadata.Map(ctx, isClient).LoadOrStore(key{}, ovsPortInfo)
	if !ok {
		return
	}
	value, ok = rawValue.(OvsPortInfo)
	return value, ok
}

// LoadAndDelete retrievs ovsPortInfo from cache and also also delete it from the cache.
func LoadAndDelete(ctx context.Context, isClient bool) (value OvsPortInfo, ok bool) {
	rawValue, ok := metadata.Map(ctx, isClient).LoadAndDelete(key{})
	if !ok {
		return
	}
	value, ok = rawValue.(OvsPortInfo)
	return value, ok
}
