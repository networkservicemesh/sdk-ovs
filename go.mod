module github.com/networkservicemesh/sdk-ovs

go 1.16

require (
	github.com/Mellanox/sriovnet v1.0.3-0.20210630121212-0453bd4b7fbc
	github.com/golang/protobuf v1.5.2
	github.com/networkservicemesh/api v1.2.1-0.20220315001249-f33f8c3f2feb
	github.com/networkservicemesh/sdk v0.5.1-0.20220328192928-739e3867a1b0
	github.com/networkservicemesh/sdk-kernel v0.0.0-20220328193242-a7872b5227b3
	github.com/networkservicemesh/sdk-sriov v0.0.0-20220328194129-659799c12eae
	github.com/ovn-org/ovn-kubernetes/go-controller v0.0.0-20210826171620-f06c53111a31
	github.com/pkg/errors v0.9.1
	github.com/vishvananda/netlink v1.1.1-0.20220118170537-d6b03fdeb845
	google.golang.org/grpc v1.42.0
	k8s.io/klog/v2 v2.40.1 // indirect
	k8s.io/utils v0.0.0-20210707171843-4b05e18ac7d9
)
