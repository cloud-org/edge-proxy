package options

import (
	"fmt"

	"github.com/spf13/pflag"
)

// BenchMarkOptions is the main settings for the edge-proxy
type BenchMarkOptions struct {
	TimeOut       int // second
	Namespace     string
	UseKubeConfig bool
	BenchType     string
}

// NewBenchmarkOptions creates a new BenchMarkOptions with a default config.
func NewBenchmarkOptions() *BenchMarkOptions {
	o := &BenchMarkOptions{
		TimeOut:   60 * 30, // second
		BenchType: "all",
	}
	return o
}

// Validate validates BenchMarkOptions
func (o *BenchMarkOptions) Validate() error {
	if o.TimeOut == 0 {
		return fmt.Errorf("timeout is zero")
	}

	return nil
}

// AddFlags returns flags for a specific edge proxy by section name
func (o *BenchMarkOptions) AddFlags(fs *pflag.FlagSet) {
	fs.IntVar(&o.TimeOut, "timeout", o.TimeOut, "bench mark timeout (second)")
	fs.StringVar(&o.Namespace, "namespace", o.Namespace, "bench mark namespace")
	fs.StringVar(&o.BenchType, "bench", o.BenchType, "bench type(all|resource|func|filter|consistency)")
	fs.BoolVar(&o.UseKubeConfig, "use-kubeconfig", o.UseKubeConfig, "use kubeconfig or not. 集群外测试使用")
}
