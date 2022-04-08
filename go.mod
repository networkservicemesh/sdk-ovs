module github.com/networkservicemesh/sdk-ovs

go 1.16

require (
	github.com/Mellanox/sriovnet v1.0.3-0.20210630121212-0453bd4b7fbc
	github.com/golang/protobuf v1.5.2
	github.com/networkservicemesh/api v1.3.0-rc.1.0.20220405210054-fbcde048efa5
	github.com/networkservicemesh/sdk v0.5.1-0.20220408124029-adfd2db62d17
	github.com/networkservicemesh/sdk-kernel v0.0.0-20220408124342-c6a45e684ec8
	github.com/networkservicemesh/sdk-sriov v0.0.0-20220408125444-7f4929ac39a1
	github.com/ovn-org/ovn-kubernetes/go-controller v0.0.0-20210826171620-f06c53111a31
	github.com/pkg/errors v0.9.1
	github.com/vishvananda/netlink v1.1.1-0.20220118170537-d6b03fdeb845
	google.golang.org/grpc v1.42.0
	k8s.io/klog/v2 v2.40.1 // indirect
	k8s.io/utils v0.0.0-20210707171843-4b05e18ac7d9
)
