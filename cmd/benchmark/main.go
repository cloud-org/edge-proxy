package main

import (
	"flag"
	"math/rand"
	"time"

	"k8s.io/apiserver/pkg/server"

	"code.aliyun.com/openyurt/edge-proxy/cmd/benchmark/app"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	cmd := app.NewCmdBenchMark(server.SetupSignalHandler())
	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
