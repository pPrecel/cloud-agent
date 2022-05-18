package serve

import (
	v1beta1_apis "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/pPrecel/cloudagent/internal/watcher"
	"github.com/pPrecel/cloudagent/pkg/agent"
	cloud_agent "github.com/pPrecel/cloudagent/pkg/agent/proto"
	"github.com/pPrecel/cloudagent/pkg/config"
	"github.com/spf13/cobra"
	googlerpc "google.golang.org/grpc"
)

func NewCmd(o *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve clouds watching.",
		Long:  "Use this command to serve an agent functionality to observe clouds you specify in the configuration file.",
		PreRunE: func(_ *cobra.Command, _ []string) error {
			return o.validate()
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return run(o)
		},
	}

	cmd.Flags().StringVarP(&o.configPath, "config-path", "c", config.ConfigPath, "Provides path to the config file.")

	return cmd
}

func run(o *options) error {
	o.Logger.Info("starting gardeners agent")

	gardenerCache := agent.NewCache[*v1beta1_apis.ShootList]()
	go func() {
		for {
			startWatcher(o, gardenerCache)
		}
	}()

	o.Logger.Debug("configuring grpc server")
	lis, err := agent.NewSocket(o.socketNetwork, o.socketAddress)
	if err != nil {
		return err
	}

	grpcServer := googlerpc.NewServer(googlerpc.EmptyServerOption{})
	agentServer := agent.NewServer(&agent.ServerOption{
		GardenerCache: gardenerCache,
		Logger:        o.Logger,
	})
	cloud_agent.RegisterAgentServer(grpcServer, agentServer)

	o.Logger.Info("starting grpc server")
	return grpcServer.Serve(lis)
}

func startWatcher(o *options, cache agent.Cache[*v1beta1_apis.ShootList]) {
	if err := watcher.NewWatcher().Start(&watcher.Options{
		Context:    o.Context,
		Logger:     o.Logger,
		Cache:      cache,
		ConfigPath: o.configPath,
	}); err != nil {
		o.Logger.Warn(err)
	}

	o.Logger.Info("configuration midyfication detected")

	o.Logger.Info("cleaning up cache")
	cache.Clean()
}
