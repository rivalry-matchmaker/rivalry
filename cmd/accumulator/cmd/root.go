package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/rivalry-matchmaker/rivalry/internal/app/backend"
	"github.com/rivalry-matchmaker/rivalry/internal/db/kv"
	"github.com/rivalry-matchmaker/rivalry/internal/db/pubsub"
	"github.com/rivalry-matchmaker/rivalry/internal/db/stream"
	"github.com/rivalry-matchmaker/rivalry/internal/dlm"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/backfill"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/customlogic"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/matches"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/tickets"
	"github.com/rivalry-matchmaker/rivalry/pkg/pb"
	"github.com/go-redis/redis/v8"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
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
		Use:   "accumulator",
		Short: "",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			matchmakerTarget, _ := cmd.Flags().GetString("matchmaker_target")
			matchmakerConn, err := grpc.Dial(matchmakerTarget, grpc.WithInsecure())
			if err != nil {
				return errors.Wrap(err, "failed to connect to assignment service")
			}
			matchmakerClient := pb.NewMatchMakerServiceClient(matchmakerConn)
			matchmakerManager := customlogic.NewMatchmakerManager(matchmakerClient)

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
			matchesManager := matches.NewManager(ticketsManager, kvStore, streamClient)
			defer func() {
				ticketsManager.Close()
				matchesManager.Close()
			}()

			backfillManagerDLM := dlm.NewRedisDLM("backfill_manager", redisOpts)
			backfillManager := backfill.NewManager(kvStore, sortedSet, set, backfillManagerDLM, ticketsManager, 1)

			profile, _ := cmd.Flags().GetString("profile")
			profilesStr, _ := cmd.Flags().GetString("profiles")
			var profiles []*pb.MatchProfile
			err = json.Unmarshal([]byte(profilesStr), &profiles)
			if err != nil {
				msg := "failed to unmarshal profiles"
				log.Err(err).Msg(msg)
				return errors.Wrap(err, msg)
			}

			maxTickets, _ := cmd.Flags().GetInt64("max_tickets")
			maxBackfill, _ := cmd.Flags().GetInt64("max_backfill")
			maxDelay, _ := cmd.Flags().GetInt64("max_delay")
			config := &backend.AccumulatorConfig{
				MaxTickets:  maxTickets,
				MaxBackfill: maxBackfill,
				MaxDelay:    time.Duration(maxDelay) * time.Millisecond,
			}

			ctx, cancel := context.WithCancel(context.Background())
			var wg sync.WaitGroup
			wg.Add(1)
			Stop = func() {
				cancel()
				wg.Wait()
			}

			accumulator := backend.NewAccumulator(ctx, profile, profiles, ticketsManager, backfillManager, matchesManager, matchmakerManager, config)

			err = accumulator.Run()
			log.Err(err).Msg("accumulator finished")
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

	rootCmd.Flags().String("profile", "", "")
	rootCmd.MarkFlagRequired("profile")
	rootCmd.Flags().String("profiles", "", "")
	rootCmd.MarkFlagRequired("profiles")

	rootCmd.Flags().String("matchmaker_target", "", "")
	rootCmd.MarkFlagRequired("matchmaker_target")

	rootCmd.Flags().Int64("max_tickets", 1000, "max number of tickets to accumulate before the match function is triggered")
	rootCmd.Flags().Int64("max_backfill", 100, "max number of backfill to submit to a match function")
	rootCmd.Flags().Int64("max_delay", time.Second.Milliseconds(), "the number of milliseconds to pass before the match function is triggered")
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
		viper.SetConfigName(".accumulator")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
