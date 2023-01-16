package cmd

import (
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/rivalry-matchmaker/rivalry/examples/matchmaker/service"
	"github.com/rivalry-matchmaker/rivalry/pkg/pb"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	homedir "github.com/mitchellh/go-homedir"
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
		Use:   "matchmaker",
		Short: "",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			address, _ := cmd.Flags().GetString("address")
			ln, err := net.Listen("tcp", address)
			if err != nil {
				log.Fatal().Msgf("cannot start tcp server on address %s: %s", address, err)
			}

			backfill, _ := cmd.Flags().GetBool("backfill")
			mm := service.NewPairsMatchmaker(backfill)
			grpcServer := grpc.NewServer()
			var wg sync.WaitGroup
			wg.Add(1)
			Stop = func() {
				grpcServer.GracefulStop()
				wg.Wait()
			}
			pb.RegisterMatchMakerServiceServer(grpcServer, mm)
			reflection.Register(grpcServer)

			log.Info().Msg("gRPC Server Serve on " + address)
			err = grpcServer.Serve(ln)
			log.Err(err).Msg("matchmaker finished")
			wg.Done()
			return err
		},
	}
	cobra.OnInitialize(initConfig)
	pf := rootCmd.PersistentFlags()
	pf.String("address", ":50051", "the address this service listens on")
	pf.Bool("backfill", false, "")
	pf.StringVar(&cfgFile, "config", "", "config file (default is $HOME/.matchmaker.yaml)")
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

		// Search config in home directory with name ".matchmaker" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".matchmaker")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
