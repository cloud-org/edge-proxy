package app

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"

	"code.aliyun.com/openyurt/edge-proxy/cmd/edge-proxy/app/config"
	"code.aliyun.com/openyurt/edge-proxy/cmd/edge-proxy/app/options"
	"code.aliyun.com/openyurt/edge-proxy/pkg/projectinfo"
	"code.aliyun.com/openyurt/edge-proxy/pkg/proxy"
	"code.aliyun.com/openyurt/edge-proxy/pkg/server"
)

// NewCmdStartEdgeProxy creates a *cobra.Command object with default parameters
func NewCmdStartEdgeProxy(stopCh <-chan struct{}) *cobra.Command {
	edgeProxyOptions := options.NewEdgeProxyOptions()

	cmd := &cobra.Command{
		Use:   projectinfo.GetProxyName(),
		Short: "Launch " + projectinfo.GetProxyName(),
		Long:  "Launch " + projectinfo.GetProxyName(),
		Run: func(cmd *cobra.Command, args []string) {
			if edgeProxyOptions.Version {
				fmt.Printf("%s: %#v\n", projectinfo.GetProxyName(), projectinfo.Get())
				return
			}
			fmt.Printf("%s version: %#v\n", projectinfo.GetProxyName(), projectinfo.Get())

			cmd.Flags().VisitAll(func(flag *pflag.Flag) {
				klog.V(1).Infof("FLAG: --%s=%q", flag.Name, flag.Value)
			})

			// 校验 proxy options 是否 ok
			if err := edgeProxyOptions.Validate(); err != nil {
				klog.Fatalf("validate options: %v", err)
			}

			// 补齐 edge-proxy 配置
			edgeProxyCfg, err := config.Complete(edgeProxyOptions)
			if err != nil {
				klog.Fatalf("complete %s configuration error, %v", projectinfo.GetProxyName(), err)
			}
			klog.Infof("%s cfg: %#+v", projectinfo.GetProxyName(), edgeProxyCfg)

			if err = Run(edgeProxyCfg, stopCh); err != nil {
				klog.Fatalf("run %s failed, %v", projectinfo.GetProxyName(), err)
			}
		},
	}

	edgeProxyOptions.AddFlags(cmd.Flags())
	return cmd
}

// Run runs the EdgeProxyConfiguration. This should never exit
func Run(cfg *config.EdgeProxyConfiguration, stopCh <-chan struct{}) error {
	trace := 1
	klog.Infof("%d. new reverse proxy handler for remote servers", trace)
	// http.Handler
	edgeProxyHandler, err := proxy.GetProxyHandler(cfg, stopCh)
	if err != nil {
		return fmt.Errorf("could not create reverse proxy handler, %w", err)
	}
	trace++

	klog.Infof(
		"%d. new %s server and begin to serve, proxy server: %s, stub server: %s",
		trace,
		projectinfo.GetProxyName(),
		cfg.EdgeProxyServerAddr,
		cfg.BindAddr,
	)
	s, err := server.NewEdgeProxyServer(cfg, edgeProxyHandler)
	if err != nil {
		return fmt.Errorf("could not create hub server, %w", err)
	}
	// 服务监听 阻塞
	s.Run()
	klog.Infof("edge proxy exited")
	return nil
}
