package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	pb "github.com/rivalry-matchmaker/rivalry/pkg/pb/api/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// matchCmd represents the match command
func newMatchCmd() *cobra.Command {
	var matchCmd = &cobra.Command{
		Use:   "match",
		Short: "",
		Long:  ``,
	}
	matchCmd.PersistentFlags().String("target", "", "gRPC connection target")
	matchCmd.MarkPersistentFlagRequired("target")
	matchCmd.AddCommand(
		newMatchRequestCmd(),
	)
	return matchCmd
}

func getCli(cmd *cobra.Command) (pb.RivalryServiceClient, error) {
	target, _ := cmd.Flags().GetString("target")

	conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC: %w", err)
	}
	return pb.NewRivalryServiceClient(conn), nil
}

// matchCmd represents the match command
func newMatchRequestCmd() *cobra.Command {
	var matchCmd = &cobra.Command{
		Use:   "request",
		Short: "A brief description of your command",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				cli        pb.RivalryServiceClient
				err        error
				tags       []string
				DA         = make(map[string]float64)
				SA         = make(map[string]string)
				playerID   string
				matchQueue string
			)

			if cli, err = getCli(cmd); err != nil {
				return err
			}

			doubleArgs, _ := cmd.Flags().GetString("doubles")
			if err = json.Unmarshal([]byte(doubleArgs), &DA); err != nil {
				return err
			}

			stringArgs, _ := cmd.Flags().GetString("strings")
			if err = json.Unmarshal([]byte(stringArgs), &SA); err != nil {
				return err
			}

			if tags, err = cmd.Flags().GetStringSlice("tags"); err != nil {
				return err
			}

			if playerID, err = cmd.Flags().GetString("player_id"); err != nil {
				return err
			}

			if matchQueue, err = cmd.Flags().GetString("match_queue"); err != nil {
				return err
			}

			if stream, err := cli.Match(context.Background(), &pb.MatchRequest{
				PlayerId:         playerID,
				MatchmakingQueue: matchQueue,
				MatchRequestData: &pb.MatchRequestData{
					Doubles: DA,
					Strings: SA,
					Tags:    tags,
				},
			}); err != nil {
				return err
			} else {
				for {
					resp, err := stream.Recv()
					if err != nil {
						return err
					}
					fmt.Println(resp)
				}
			}
			return nil
		},
	}
	matchCmd.Flags().String("player_id", "", "The id of the player wishing to match")
	matchCmd.MarkFlagRequired("player_id")
	matchCmd.Flags().String("match_queue", "default", "The queue the player is wishing to match on")
	matchCmd.Flags().String("doubles", "{}", "The doubles match request parameter")
	matchCmd.Flags().String("strings", "{}", "The strings match request parameter")
	matchCmd.Flags().StringSlice("tags", []string{"1v1"}, "The tags match request parameter")
	// TODO Party, RTTs
	return matchCmd
}
