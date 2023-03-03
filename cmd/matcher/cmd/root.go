package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/go-redis/redis/v8"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/rivalry-matchmaker/rivalry/internal/app/backend"
	"github.com/rivalry-matchmaker/rivalry/internal/db/kv"
	"github.com/rivalry-matchmaker/rivalry/internal/db/pubsub"
	"github.com/rivalry-matchmaker/rivalry/internal/db/stream"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/customlogic"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/matches"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/tickets"
	pb "github.com/rivalry-matchmaker/rivalry/pkg/pb/api/v1"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

var (
	cfgFile string
	// Stop is a function that can be called to stop this cmd
	Stop func()
)

// NewRootCmd instantiates the command line root command
func NewRootCmd() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:   "matcher",
		Short: "",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Matchmaking Queue
			queue, _ := cmd.Flags().GetString("queue")

			// Redis
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

			// Nats
			natsAddr, _ := cmd.Flags().GetString("nats_addr")
			streamClient := stream.NewNATS(natsAddr, "om")
			pubsubClient := pubsub.NewNATS(natsAddr)

			// Ticket Manager
			ticketsManager := tickets.NewManager(kvStore, streamClient, pubsubClient)
			matchesManager := matches.NewManager(ticketsManager, kvStore, streamClient)
			defer func() {
				ticketsManager.Close()
				matchesManager.Close()
			}()

			ctx, cancel := context.WithCancel(context.Background())
			var wg sync.WaitGroup
			wg.Add(1)
			Stop = func() {
				cancel()
				wg.Wait()
			}

			matchmakerTarget, _ := cmd.Flags().GetString("matchmaker_target")
			matchmakerConn, err := grpc.Dial(matchmakerTarget, grpc.WithInsecure())
			if err != nil {
				return errors.Wrap(err, "failed to connect to assignment service")
			}
			matchmakerClient := pb.NewMatchMakerServiceClient(matchmakerConn)
			matchmakerManager := customlogic.NewMatchmakerManager(matchmakerClient)

			matcher := backend.NewMatcher(ctx, queue, ticketsManager, matchesManager, matchmakerManager)

			log.Info().Str("service", cmd.Use).Str("queue", queue).Msg("Running")
			err = matcher.Run()
			log.Err(err).Msg("matcher finished")
			wg.Done()
			return err
		},
	}
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.accumulator.yaml)")

	rootCmd.Flags().Bool("redis_unix", false, "")
	rootCmd.Flags().String("redis_addr", "", "")
	rootCmd.MarkFlagRequired("redis_addr")

	rootCmd.Flags().String("nats_addr", "", "")
	rootCmd.MarkFlagRequired("nats_addr")

	rootCmd.Flags().String("queue", "default", "the matchmaking queue")

	rootCmd.Flags().String("matchmaker_target", "", "")
	rootCmd.MarkFlagRequired("matchmaker_target")

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

		// Search config in home directory with name ".accumulator" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".matcher")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
