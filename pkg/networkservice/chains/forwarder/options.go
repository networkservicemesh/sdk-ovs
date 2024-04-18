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
	"net/url"
	"time"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk-ovs/pkg/networkservice/mechanisms/vxlan"
)

type forwarderOptions struct {
	name                             string
	bridgeName                       string
	authorizeServer                  networkservice.NetworkServiceServer
	authorizeMonitorConnectionServer networkservice.MonitorConnectionServer
	resourcePoolServer               networkservice.NetworkServiceServer
	resourcePoolClient               networkservice.NetworkServiceClient
	clientURL                        *url.URL
	dialTimeout                      time.Duration
	vxlanOpts                        []vxlan.Option
	dialOpts                         []grpc.DialOption
}

// Option is an option pattern for forwarder chain elements
type Option func(o *forwarderOptions)

// WithName sets forwarder name
func WithName(name string) Option {
	return func(o *forwarderOptions) {
		o.name = name
	}
}

// WithBridgeName sets bridge name
func WithBridgeName(bridgeName string) Option {
	return func(o *forwarderOptions) {
		o.bridgeName = bridgeName
	}
}

// WithAuthorizeServer sets authorization server chain element
func WithAuthorizeServer(authorizeServer networkservice.NetworkServiceServer) Option {
	if authorizeServer == nil {
		panic("Authorize server cannot be nil")
	}
	return func(o *forwarderOptions) {
		o.authorizeServer = authorizeServer
	}
}

// WithAuthorizeMonitorConnectionServer sets authorization server chain element
func WithAuthorizeMonitorConnectionServer(authorizeMonitorConnectionServer networkservice.MonitorConnectionServer) Option {
	if authorizeMonitorConnectionServer == nil {
		panic("Authorize monitor server cannot be nil")
	}
	return func(o *forwarderOptions) {
		o.authorizeMonitorConnectionServer = authorizeMonitorConnectionServer
	}
}

// WithResourcePoolServer sets resource pool server
func WithResourcePoolServer(resourcePoolServer networkservice.NetworkServiceServer) Option {
	if resourcePoolServer == nil {
		panic("Authorize server cannot be nil")
	}
	return func(o *forwarderOptions) {
		o.resourcePoolServer = resourcePoolServer
	}
}

// WithResourcePoolClient sets resource pool client
func WithResourcePoolClient(resourcePoolClient networkservice.NetworkServiceClient) Option {
	if resourcePoolClient == nil {
		panic("Authorize server cannot be nil")
	}
	return func(o *forwarderOptions) {
		o.resourcePoolClient = resourcePoolClient
	}
}

// WithClientURL sets clientURL.
func WithClientURL(clientURL *url.URL) Option {
	return func(c *forwarderOptions) {
		c.clientURL = clientURL
	}
}

// WithDialTimeout sets dial timeout for the client
func WithDialTimeout(dialTimeout time.Duration) Option {
	return func(o *forwarderOptions) {
		o.dialTimeout = dialTimeout
	}
}

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
