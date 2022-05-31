package options

import (
	"fmt"

	"github.com/spf13/pflag"
)

// EdgeProxyOptions is the main settings for the edge-proxy
type EdgeProxyOptions struct {
	ServerAddr    string
	NodeName      string
	JoinToken     string
	DiskCachePath string
	Version       bool
}

// NewEdgeProxyOptions creates a new EdgeProxyOptions with a default config.
func NewEdgeProxyOptions() *EdgeProxyOptions {
	o := &EdgeProxyOptions{
		DiskCachePath: "/etc/kubernetes/cache/",
	}
	return o
}

// Validate validates EdgeProxyOptions
func (options *EdgeProxyOptions) Validate() error {
	if len(options.NodeName) == 0 {
		return fmt.Errorf("node name is empty")
	}

	if len(options.ServerAddr) == 0 {
		return fmt.Errorf("server-address is empty")
	}

	if len(options.JoinToken) == 0 {
		return fmt.Errorf("join-token is empty")
	}

	return nil
}

// AddFlags returns flags for a specific yurthub by section name
func (o *EdgeProxyOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.ServerAddr, "server-addr", o.ServerAddr, "the address of Kubernetes kube-apiserver,the format is: \"server1,server2,...\"")
	fs.StringVar(&o.NodeName, "node-name", o.NodeName, "the name of node that runs hub agent")
	fs.StringVar(&o.JoinToken, "join-token", o.JoinToken, "the Join token for bootstrapping hub agent when --cert-mgr-mode=hubself.")
	fs.BoolVar(&o.Version, "version", o.Version, "print the version information.")
	fs.StringVar(&o.DiskCachePath, "disk-cache-path", o.DiskCachePath, "the path for kubernetes to storage metadata")
}
