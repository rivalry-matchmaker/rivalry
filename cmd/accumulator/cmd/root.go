package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/rivalry-matchmaker/rivalry/internal/app/backend"
	"github.com/rivalry-matchmaker/rivalry/internal/db/kv"
	"github.com/rivalry-matchmaker/rivalry/internal/db/pubsub"
	"github.com/rivalry-matchmaker/rivalry/internal/db/stream"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/tickets"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
			defer func() {
				ticketsManager.Close()
			}()

			// Accumulator Config
			maxTickets, _ := cmd.Flags().GetInt64("max_tickets")
			maxDelay, _ := cmd.Flags().GetInt64("max_delay")
			config := &backend.AccumulatorConfig{
				MaxTickets: maxTickets,
				MaxDelay:   time.Duration(maxDelay) * time.Millisecond,
			}

			ctx, cancel := context.WithCancel(context.Background())
			var wg sync.WaitGroup
			wg.Add(1)
			Stop = func() {
				cancel()
				wg.Wait()
			}

			accumulator := backend.NewAccumulator(ctx, queue, ticketsManager, config)

			log.Info().Str("service", cmd.Use).Str("queue", queue).Msg("Running")
			err := accumulator.Run()
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

	rootCmd.Flags().String("queue", "default", "the matchmaking queue")

	rootCmd.Flags().Int64("max_tickets", 1000, "max number of tickets to accumulate before the match function is triggered")
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
