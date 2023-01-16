package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rivalry-matchmaker/rivalry/pkg/pb"
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
		newMatchStatusCmd(),
	)
	return matchCmd
}

func getCli(cmd *cobra.Command) (pb.FrontendServiceClient, error) {
	target, _ := cmd.Flags().GetString("target")

	conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC: %w", err)
	}
	return pb.NewFrontendServiceClient(conn), nil
}

// matchCmd represents the match command
func newMatchRequestCmd() *cobra.Command {
	var matchCmd = &cobra.Command{
		Use:   "request",
		Short: "A brief description of your command",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				cli  pb.FrontendServiceClient
				err  error
				tags []string
				DA   = make(map[string]float64)
				SA   = make(map[string]string)
			)

			if cli, err = getCli(cmd); err != nil {
				return err
			}

			doubleArgs, _ := cmd.Flags().GetString("double_args")
			if err = json.Unmarshal([]byte(doubleArgs), &DA); err != nil {
				return err
			}

			stringArgs, _ := cmd.Flags().GetString("string_args")
			if err = json.Unmarshal([]byte(stringArgs), &SA); err != nil {
				return err
			}

			if tags, err = cmd.Flags().GetStringSlice("tags"); err != nil {
				return err
			}

			if resp, err := cli.CreateTicket(context.Background(), &pb.CreateTicketRequest{
				Ticket: &pb.Ticket{
					SearchFields: &pb.SearchFields{
						DoubleArgs: DA,
						StringArgs: SA,
						Tags:       tags,
					},
				},
			}); err != nil {
				return err
			} else {
				fmt.Println(resp)
			}
			return nil
		},
	}
	matchCmd.Flags().String("double_args", "{}", "The double args match request parameter")
	matchCmd.Flags().String("string_args", "{}", "The string args match request parameter")
	matchCmd.Flags().StringSlice("tags", []string{"1v1"}, "The tags match request parameter")
	return matchCmd
}

func newMatchStatusCmd() *cobra.Command {
	var matchCmd = &cobra.Command{
		Use:   "status",
		Short: "A brief description of your command",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				cli pb.FrontendServiceClient
				err error
			)
			if cli, err = getCli(cmd); err != nil {
				return err
			}
			ticketID, _ := cmd.Flags().GetString("ticket_id")
			stream, err := cli.WatchAssignments(context.Background(), &pb.WatchAssignmentsRequest{TicketId: ticketID})
			if err != nil {
				return err
			}

			for {
				resp, err := stream.Recv()
				if err != nil {
					return err
				}
				fmt.Println(resp)
			}
		},
	}
	matchCmd.Flags().String("ticket_id", "", "The ticket id")
	return matchCmd
}
