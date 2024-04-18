// Copyright (c) 2021-2024 Nordix Foundation.
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

// Package forwarder provides interpose endpoint implementation for ovs forwarder
// which provides kernel and smartnic endpoints
package forwarder

import (
	"context"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	vxlanmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/connectioncontextkernel"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/inject"
	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/resourcepool"
	sriovtokens "github.com/networkservicemesh/sdk-sriov/pkg/tools/tokens"
	registryclient "github.com/networkservicemesh/sdk/pkg/registry/chains/client"
	registryrecvfd "github.com/networkservicemesh/sdk/pkg/registry/common/recvfd"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/config"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/discover"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/filtermechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/recvfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanismtranslation"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/roundrobin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/switchcase"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	authmonitor "github.com/networkservicemesh/sdk/pkg/tools/monitorconnection/authorize"
	"github.com/networkservicemesh/sdk/pkg/tools/token"

	"github.com/networkservicemesh/sdk-ovs/pkg/networkservice/l2ovsconnect"
	"github.com/networkservicemesh/sdk-ovs/pkg/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk-ovs/pkg/networkservice/mechanisms/vlan"
	"github.com/networkservicemesh/sdk-ovs/pkg/networkservice/mechanisms/vxlan"
	ovsutil "github.com/networkservicemesh/sdk-ovs/pkg/tools/utils"
)

type ovsConnectNSServer struct {
	endpoint.Endpoint
}

// NewSriovServer - returns sriov implementation of the ovsconnectns network service
func NewSriovServer(ctx context.Context, tokenGenerator token.GeneratorFunc, tunnelIPCidr net.IP,
	pciPool resourcepool.PCIPool, resourcePool resourcepool.ResourcePool, sriovConfig *config.Config,
	l2Connections map[string]*ovsutil.L2ConnectionPoint, options ...Option,
) (endpoint.Endpoint, error) {
	resourceLock := &sync.Mutex{}
	resourcePoolClient := resourcepool.NewClient(sriov.KernelDriver, resourceLock, pciPool, resourcePool, sriovConfig)
	resourcePoolServer := resourcepool.NewServer(sriov.KernelDriver, resourceLock, pciPool, resourcePool, sriovConfig)
	options = append(options, WithResourcePoolServer(resourcePoolServer), WithResourcePoolClient(resourcePoolClient))

	return newEndPoint(ctx, tokenGenerator, tunnelIPCidr, l2Connections, options...)
}

func newEndPoint(ctx context.Context, tokenGenerator token.GeneratorFunc, tunnelIPCidr net.IP,
	l2Connections map[string]*ovsutil.L2ConnectionPoint, options ...Option,
) (endpoint.Endpoint, error) {
	opts := &forwarderOptions{
		name:                             "forwarder-ovs-" + uuid.New().String(),
		bridgeName:                       "br-nsm",
		authorizeServer:                  authorize.NewServer(authorize.Any()),
		authorizeMonitorConnectionServer: authmonitor.NewMonitorConnectionServer(authmonitor.Any()),
		resourcePoolServer:               null.NewServer(),
		resourcePoolClient:               null.NewClient(),
		clientURL:                        &url.URL{Scheme: "unix", Host: "connect.to.socket"},
		dialTimeout:                      time.Millisecond * 200,
	}
	for _, opt := range options {
		opt(opts)
	}
	tunnelIP, err := ovsutil.ParseTunnelIP(tunnelIPCidr)
	if err != nil {
		return nil, err
	}
	err = ovsutil.ConfigureOvS(ctx, l2Connections, opts.bridgeName)
	if err != nil {
		return nil, err
	}

	parentIfMutex := &sync.Mutex{}
	parentIfRefCount := make(map[string]int)

	vxlanInterfacesMutex := &sync.Mutex{}
	vxlanInterfaces := make(map[string]int)
	rv := &ovsConnectNSServer{}

	nseClient := registryclient.NewNetworkServiceEndpointRegistryClient(ctx,
		registryclient.WithClientURL(opts.clientURL),
		registryclient.WithNSEAdditionalFunctionality(registryrecvfd.NewNetworkServiceEndpointRegistryClient()),
		registryclient.WithDialOptions(opts.dialOpts...),
	)
	nsClient := registryclient.NewNetworkServiceRegistryClient(ctx,
		registryclient.WithClientURL(opts.clientURL),
		registryclient.WithDialOptions(opts.dialOpts...))

	additionalFunctionality := []networkservice.NetworkServiceServer{
		metadata.NewServer(),
		recvfd.NewServer(),
		sendfd.NewServer(),
		discover.NewServer(nsClient, nseClient),
		roundrobin.NewServer(),
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			kernelmech.MECHANISM: switchcase.NewServer(
				&switchcase.ServerCase{
					Condition: func(_ context.Context, conn *networkservice.Connection) bool {
						return sriovtokens.IsTokenID(kernelmech.ToMechanism(conn.GetMechanism()).GetDeviceTokenID())
					},
					Server: chain.NewNetworkServiceServer(
						opts.resourcePoolServer,
						kernel.NewSmartVFServer(opts.bridgeName, parentIfMutex, parentIfRefCount),
					),
				},
				&switchcase.ServerCase{
					Condition: switchcase.Default,
					Server:    kernel.NewVethServer(opts.bridgeName, parentIfMutex, parentIfRefCount),
				},
			),
			vxlanmech.MECHANISM: vxlan.NewServer(tunnelIP, opts.bridgeName, vxlanInterfacesMutex, vxlanInterfaces, opts.vxlanOpts...),
		}),
		inject.NewServer(),
		connectioncontextkernel.NewServer(),
		connect.NewServer(
			client.NewClient(ctx,
				client.WithoutRefresh(),
				client.WithName(opts.name),
				client.WithDialOptions(opts.dialOpts...),
				client.WithDialTimeout(opts.dialTimeout),
				client.WithAdditionalFunctionality(
					mechanismtranslation.NewClient(),
					l2ovsconnect.NewClient(opts.bridgeName),
					connectioncontextkernel.NewClient(),
					inject.NewClient(),
					// mechanisms
					kernel.NewClient(opts.bridgeName, parentIfMutex, parentIfRefCount),
					opts.resourcePoolClient,
					vxlan.NewClient(tunnelIP, opts.bridgeName, vxlanInterfacesMutex, vxlanInterfaces, opts.vxlanOpts...),
					vlan.NewClient(opts.bridgeName, l2Connections),
					filtermechanisms.NewClient(),
					recvfd.NewClient(),
					sendfd.NewClient(),
				),
			),
		),
	}

	rv.Endpoint = endpoint.NewServer(ctx, tokenGenerator,
		endpoint.WithName(opts.name),
		endpoint.WithAuthorizeServer(opts.authorizeServer),
		endpoint.WithAuthorizeMonitorConnectionServer(opts.authorizeMonitorConnectionServer),
		endpoint.WithAdditionalFunctionality(additionalFunctionality...))

	return rv, nil
}

// NewKernelServer - returns kernel implementation of the ovsconnectns network service
func NewKernelServer(ctx context.Context, tokenGenerator token.GeneratorFunc, tunnelIPCidr net.IP,
	l2Connections map[string]*ovsutil.L2ConnectionPoint, options ...Option,
) (endpoint.Endpoint, error) {
	return newEndPoint(ctx, tokenGenerator, tunnelIPCidr, l2Connections, options...)
}
