package cmd

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/rivalry-matchmaker/rivalry/internal/app/frontend"
	"github.com/rivalry-matchmaker/rivalry/internal/db/kv"
	"github.com/rivalry-matchmaker/rivalry/internal/db/pubsub"
	"github.com/rivalry-matchmaker/rivalry/internal/db/stream"
	"github.com/rivalry-matchmaker/rivalry/internal/dlm"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/backfill"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/customlogic"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/filter"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/tickets"
	"github.com/rivalry-matchmaker/rivalry/pkg/pb"
	"github.com/go-redis/redis/v8"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
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
			sortedSet := kvStore.(kv.SortedSet)
			set := kvStore.(kv.Set)

			natsAddr, _ := cmd.Flags().GetString("nats_addr")
			streamClient := stream.NewNATS(natsAddr, "om")

			pubsubClient := pubsub.NewNATS(natsAddr)

			ticketsManager := tickets.NewManager(kvStore, streamClient, pubsubClient)

			var validationClient pb.ValidationServiceClient
			validationTarget, _ := cmd.Flags().GetString("validation_target")
			if len(validationTarget) > 0 {
				validationConn, err := grpc.Dial(validationTarget, grpc.WithInsecure())
				if err != nil {
					return errors.Wrap(err, "failed to connect to assignment service")
				}
				validationClient = pb.NewValidationServiceClient(validationConn)
			}

			var dataClient pb.DataServiceClient
			dataTarget, _ := cmd.Flags().GetString("data_target")
			if len(dataTarget) > 0 {
				dataConn, err := grpc.Dial(dataTarget, grpc.WithInsecure())
				if err != nil {
					return errors.Wrap(err, "failed to connect to assignment service")
				}
				dataClient = pb.NewDataServiceClient(dataConn)
			}
			log.Trace().Msg("dataClient")

			customlogicManager := customlogic.NewFrontendManager(validationClient, dataClient)

			profilesStr, _ := cmd.Flags().GetString("profiles")
			var profiles []*pb.MatchProfile
			err = json.Unmarshal([]byte(profilesStr), &profiles)
			if err != nil {
				msg := "failed to unmarshal profiles"
				log.Err(err).Msg(msg)
				return errors.Wrap(err, msg)
			}
			filterManager := filter.NewManager(profiles)

			backfillManagerDLM := dlm.NewRedisDLM("backfill_manager", redisOpts)
			backfillManager := backfill.NewManager(kvStore, sortedSet, set, backfillManagerDLM, ticketsManager, 1)

			frontendService := frontend.NewService(ticketsManager, backfillManager, customlogicManager, filterManager)
			defer frontendService.Close()

			grpcServer := grpc.NewServer()
			var wg sync.WaitGroup
			wg.Add(1)
			Stop = func() {
				grpcServer.GracefulStop()
				wg.Wait()
			}
			pb.RegisterFrontendServiceServer(grpcServer, frontendService)
			reflection.Register(grpcServer)

			log.Info().Msg("gRPC Server Serve on " + address)
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

	rootCmd.Flags().String("profiles", "", "")
	rootCmd.MarkFlagRequired("profiles")

	rootCmd.Flags().String("validation_target", "", "")

	rootCmd.Flags().String("data_target", "", "")

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
