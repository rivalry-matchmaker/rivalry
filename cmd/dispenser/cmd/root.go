package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/rivalry-matchmaker/rivalry/internal/app/backend"
	"github.com/rivalry-matchmaker/rivalry/internal/db/kv"
	"github.com/rivalry-matchmaker/rivalry/internal/db/pubsub"
	"github.com/rivalry-matchmaker/rivalry/internal/db/stream"
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
		Use:   "dispenser",
		Short: "",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			assignmentTarget, _ := cmd.Flags().GetString("assignment_target")
			assignmentConn, err := grpc.Dial(assignmentTarget, grpc.WithInsecure())
			if err != nil {
				return errors.Wrap(err, "failed to connect to assignment service")
			}
			assignmentClient := pb.NewAssignmentServiceClient(assignmentConn)
			assignmentManager := customlogic.NewAssignmentManager(assignmentClient)

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
			matchesManager := matches.NewManager(ticketsManager, kvStore, streamClient)
			defer func() {
				ticketsManager.Close()
				matchesManager.Close()
			}()

			dispenser := backend.NewDispenser(matchesManager, ticketsManager, assignmentManager)

			ctx, cancel := context.WithCancel(context.Background())
			var wg sync.WaitGroup
			wg.Add(1)
			Stop = func() {
				cancel()
				wg.Wait()
			}
			err = dispenser.Run(ctx)
			log.Err(err).Msg("dispenser finished")
			wg.Done()
			return err
		},
	}
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.dispenser.yaml)")

	rootCmd.Flags().Bool("redis_unix", false, "")
	rootCmd.Flags().String("redis_addr", "", "")
	rootCmd.MarkFlagRequired("redis_addr")

	rootCmd.Flags().String("nats_addr", "", "")
	rootCmd.MarkFlagRequired("nats_addr")

	rootCmd.Flags().String("assignment_target", "", "")
	rootCmd.MarkFlagRequired("assignment_target")

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

		// Search config in home directory with name ".dispenser" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".dispenser")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
