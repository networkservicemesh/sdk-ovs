module github.com/networkservicemesh/sdk-ovs

go 1.16

require (
	github.com/Mellanox/sriovnet v1.0.3-0.20210630121212-0453bd4b7fbc
	github.com/golang/protobuf v1.5.2
	github.com/networkservicemesh/api v1.1.2-0.20220119092736-21eda250c390
	github.com/networkservicemesh/sdk v0.5.1-0.20220209093015-129dfffd3ca9
	github.com/networkservicemesh/sdk-kernel v0.0.0-20220209093812-614e75d1c121
	github.com/networkservicemesh/sdk-sriov v0.0.0-20220209094418-fbc8190a8fb3
	github.com/ovn-org/ovn-kubernetes/go-controller v0.0.0-20210826171620-f06c53111a31
	github.com/pkg/errors v0.9.1
	github.com/vishvananda/netlink v1.1.1-0.20220118170537-d6b03fdeb845
	google.golang.org/grpc v1.42.0
	k8s.io/klog/v2 v2.40.1 // indirect
	k8s.io/utils v0.0.0-20210707171843-4b05e18ac7d9
)
