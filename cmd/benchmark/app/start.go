package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"code.aliyun.com/openyurt/edge-proxy/pkg/benchmark"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"

	"code.aliyun.com/openyurt/edge-proxy/cmd/benchmark/app/options"
	"code.aliyun.com/openyurt/edge-proxy/pkg/projectinfo"
)

const (
	// component name
	componentBenchMarkTask = "benchmark"
)

// NewCmdBenchMark creates a *cobra.Command object with default parameters
func NewCmdBenchMark(stopCh <-chan struct{}) *cobra.Command {
	benchMarkOptions := options.NewBenchmarkOptions()

	cleanFlagSet := pflag.NewFlagSet(componentBenchMarkTask, pflag.ContinueOnError)

	cmd := &cobra.Command{

		Use:                componentBenchMarkTask,
		Long:               fmt.Sprintf("The benchmark program from %s.", projectinfo.GetProxyName()),
		DisableFlagParsing: true,
		Run: func(cmd *cobra.Command, args []string) {
			// initial flag parse, since we disable cobra's flag parsing
			if err := cleanFlagSet.Parse(args); err != nil {
				cmd.Usage()
				klog.Fatal(err)
			}

			// check if there are non-flag arguments in the command line
			cmds := cleanFlagSet.Args()
			if len(cmds) > 0 {
				cmd.Usage()
				klog.Fatalf("unknown command: %s", cmds[0])
			}

			// short-circuit on help
			help, err := cleanFlagSet.GetBool("help")
			if err != nil {
				klog.Fatal(`"help" flag is non-bool, programmer error, please correct`)
			}
			if help {
				cmd.Help()
				return
			}

			klog.V(2).Infof("BenchMark Config: %#v", *benchMarkOptions)
			klog.Infof("Version:%#v", projectinfo.Get())

			// validate the initial fetch-task flags
			if err := benchMarkOptions.Validate(); err != nil {
				klog.Fatal(err)
			}

			shutdownHandler := make(chan os.Signal, 2)
			var shutdownSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(benchMarkOptions.TimeOut)*time.Second)
			defer cancel()
			signal.Notify(shutdownHandler, shutdownSignals...)
			go func() {

				<-shutdownHandler

				cancel()
				os.Exit(1) // second signal. Exit directly.
			}()

			var runError error

			runError = Run(ctx, benchMarkOptions)
			if runError != nil {
				klog.Fatal(runError)
			}
		},
	}

	// keep cleanFlagSet separate, so Cobra doesn't pollute it with the global flags
	benchMarkOptions.AddFlags(cleanFlagSet)
	// DELETE BY zhangjie
	options.AddGlobalFlags(cleanFlagSet)
	cleanFlagSet.BoolP("help", "h", false, fmt.Sprintf("help for %s", cmd.Name()))

	// ugly, but necessary, because Cobra's default UsageFunc and HelpFunc pollute the flagset with global flags
	const usageFmt = "Usage:\n  %s\n\nFlags:\n%s"
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine(), cleanFlagSet.FlagUsagesWrapped(2))
		return nil
	})
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine(), cleanFlagSet.FlagUsagesWrapped(2))
	})

	return cmd
}

// Run runs the EdgeProxyConfiguration. This should never exit
func Run(ctx context.Context, markOptions *options.BenchMarkOptions) error {

	// sleep until edge-proxy ready
	klog.Infof("Sleep 1 minute to wait edge-proxy ready")
	time.Sleep(time.Minute)

	b, err := benchmark.NewBenchMark(markOptions)
	if err != nil {
		klog.Fatalf("NewBenchMark error %v", err)
	}

	b.Run(ctx)

	for {
		time.Sleep(time.Second)
	}
}
