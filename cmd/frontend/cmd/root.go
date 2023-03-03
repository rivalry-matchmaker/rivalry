package cmd

import (
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/go-redis/redis/v8"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/rivalry-matchmaker/rivalry/internal/app/frontend"
	"github.com/rivalry-matchmaker/rivalry/internal/db/kv"
	"github.com/rivalry-matchmaker/rivalry/internal/db/pubsub"
	"github.com/rivalry-matchmaker/rivalry/internal/db/stream"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/tickets"
	pb "github.com/rivalry-matchmaker/rivalry/pkg/pb/api/v1"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	cfgFile string
	// Stop is a function that can be called to stop this cmd
	Stop func()
)

// NewRootCmd instantiates the command line root command
func NewRootCmd() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:   "frontend",
		Short: "",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			address, _ := cmd.Flags().GetString("address")
			ln, err := net.Listen("tcp", address)
			if err != nil {
				log.Fatal().Msgf("cannot start tcp server on address %s: %s", address, err)
			}

			redisAddr, _ := cmd.Flags().GetString("redis_addr")
			network := "tcp"
			if isUnix, _ := cmd.Flags().GetBool("redis_unix"); isUnix {
				network = "unix"
			}
			redisOpts := &redis.Options{
				Network: network,
				Addr:    redisAddr,
			}
			kvStore := kv.NewRedis("om", redisOpts)

			natsAddr, _ := cmd.Flags().GetString("nats_addr")
			streamClient := stream.NewNATS(natsAddr, "om")

			pubsubClient := pubsub.NewNATS(natsAddr)

			ticketsManager := tickets.NewManager(kvStore, streamClient, pubsubClient)

			queues, _ := cmd.Flags().GetStringSlice("queues")

			frontendService := frontend.NewService(queues, ticketsManager)
			defer frontendService.Close()

			grpcServer := grpc.NewServer()
			var wg sync.WaitGroup
			wg.Add(1)
			Stop = func() {
				grpcServer.GracefulStop()
				wg.Wait()
			}
			pb.RegisterRivalryServiceServer(grpcServer, frontendService)
			reflection.Register(grpcServer)

			log.Info().Str("service", cmd.Use).Str("address", address).Msg("gRPC Server Serve")
			err = grpcServer.Serve(ln)
			log.Err(err).Msg("frontend finished")
			wg.Done()
			return err
		},
	}
	cobra.OnInitialize(initConfig)
	pf := rootCmd.PersistentFlags()
	pf.String("address", ":50051", "the address this service listens on")
	pf.StringVar(&cfgFile, "config", "", "config file (default is $HOME/.frontend.yaml)")

	rootCmd.Flags().Bool("redis_unix", false, "")
	rootCmd.Flags().String("redis_addr", "", "")
	rootCmd.MarkFlagRequired("redis_addr")

	rootCmd.Flags().String("nats_addr", "", "")
	rootCmd.MarkFlagRequired("nats_addr")

	rootCmd.Flags().StringSlice("queues", []string{"default"}, "the list of available matchmaking queues")

	return rootCmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd := NewRootCmd()
	cobra.CheckErr(rootCmd.Execute())
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".frontend" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".frontend")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
