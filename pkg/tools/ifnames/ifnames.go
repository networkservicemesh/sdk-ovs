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

package ifnames

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type key struct{}

type OvsPortInfo struct {
	PortName        string
	PortNo          int
	IsTunnelPort    bool
	IsVfRepresentor bool
	VNI             uint32
}

func Store(ctx context.Context, isClient bool, ovsPortInfo OvsPortInfo) {
	metadata.Map(ctx, isClient).Store(key{}, ovsPortInfo)
}

func Delete(ctx context.Context, isClient bool) {
	metadata.Map(ctx, isClient).Delete(key{})
}

func Load(ctx context.Context, isClient bool) (value OvsPortInfo, ok bool) {
	rawValue, ok := metadata.Map(ctx, isClient).Load(key{})
	if !ok {
		return
	}
	value, ok = rawValue.(OvsPortInfo)
	return value, ok
}

func LoadOrStore(ctx context.Context, isClient bool, ovsPortInfo OvsPortInfo) (value OvsPortInfo, ok bool) {
	rawValue, ok := metadata.Map(ctx, isClient).LoadOrStore(key{}, ovsPortInfo)
	if !ok {
		return
	}
	value, ok = rawValue.(OvsPortInfo)
	return value, ok
}

func LoadAndDelete(ctx context.Context, isClient bool) (value OvsPortInfo, ok bool) {
	rawValue, ok := metadata.Map(ctx, isClient).LoadAndDelete(key{})
	if !ok {
		return
	}
	value, ok = rawValue.(OvsPortInfo)
	return value, ok
}
