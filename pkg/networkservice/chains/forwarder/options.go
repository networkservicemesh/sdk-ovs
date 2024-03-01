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

//go:build linux
// +build linux

package forwarder

import (
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk-ovs/pkg/networkservice/mechanisms/vxlan"
)

type forwarderOptions struct {
	vxlanOpts []vxlan.Option
	dialOpts  []grpc.DialOption
}

// Option is an option pattern for forwarder chain elements
type Option func(o *forwarderOptions)

// WithVxlanOptions sets vxlan option
func WithVxlanOptions(opts ...vxlan.Option) Option {
	return func(o *forwarderOptions) {
		o.vxlanOpts = opts
	}
}

// WithDialOptions sets dial options
func WithDialOptions(opts ...grpc.DialOption) Option {
	return func(o *forwarderOptions) {
		o.dialOpts = opts
	}
}
