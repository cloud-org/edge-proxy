package main

import (
	"flag"
	"math/rand"
	"time"

	"k8s.io/apiserver/pkg/server"

	"code.aliyun.com/openyurt/edge-proxy/cmd/edge-proxy/app"
	_ "code.aliyun.com/openyurt/edge-proxy/pkg/proxy/sample"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	cmd := app.NewCmdStartEdgeProxy(server.SetupSignalHandler())
	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
