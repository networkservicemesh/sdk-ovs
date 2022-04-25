module github.com/networkservicemesh/sdk-ovs

go 1.16

require (
	github.com/Mellanox/sriovnet v1.0.3-0.20210630121212-0453bd4b7fbc
	github.com/golang/protobuf v1.5.2
	github.com/networkservicemesh/api v1.3.0-rc.1.0.20220405210054-fbcde048efa5
	github.com/networkservicemesh/sdk v0.5.1-0.20220425233641-318f409ee273
	github.com/networkservicemesh/sdk-kernel v0.0.0-20220425234003-8af7ac522592
	github.com/networkservicemesh/sdk-sriov v0.0.0-20220425234601-89c279b84fcd
	github.com/ovn-org/ovn-kubernetes/go-controller v0.0.0-20210826171620-f06c53111a31
	github.com/pkg/errors v0.9.1
	github.com/vishvananda/netlink v1.1.1-0.20220118170537-d6b03fdeb845
	google.golang.org/grpc v1.42.0
	k8s.io/klog/v2 v2.40.1 // indirect
	k8s.io/utils v0.0.0-20210707171843-4b05e18ac7d9
)
