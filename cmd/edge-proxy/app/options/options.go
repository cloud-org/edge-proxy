package options

import (
	"fmt"

	"github.com/spf13/pflag"
)

// EdgeProxyOptions is the main settings for the edge-proxy
type EdgeProxyOptions struct {
	ServerAddr          string // kube api-server addr
	DiskCachePath       string // 磁盘缓存路径
	Version             bool
	EnableSampleHandler bool
	UseKubeConfig       bool
}

// NewEdgeProxyOptions creates a new EdgeProxyOptions with a default config.
func NewEdgeProxyOptions() *EdgeProxyOptions {
	o := &EdgeProxyOptions{
		DiskCachePath:       "/etc/kubernetes/cache/",
		EnableSampleHandler: false,
	}
	return o
}

// Validate validates EdgeProxyOptions
func (o *EdgeProxyOptions) Validate() error {
	if len(o.ServerAddr) == 0 {
		return fmt.Errorf("server-address is empty")
	}

	return nil
}

// AddFlags returns flags for a specific edge proxy by section name
func (o *EdgeProxyOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.ServerAddr, "server-addr", o.ServerAddr, "the address of Kubernetes kube-apiserver,the format is: \"server1,server2,...\"")
	fs.BoolVar(&o.Version, "version", o.Version, "print the version information.")
	fs.BoolVar(&o.EnableSampleHandler, "enable-sample-handler", o.EnableSampleHandler, "enable sample handler or not.")
	fs.BoolVar(&o.UseKubeConfig, "use-kubeconfig", o.UseKubeConfig, "use kubeconfig or not. 集群外测试使用")
	fs.StringVar(&o.DiskCachePath, "disk-cache-path", o.DiskCachePath, "the path for kubernetes to storage metadata")
}
