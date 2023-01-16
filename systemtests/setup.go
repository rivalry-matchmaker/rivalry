package systemtests

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stvp/tempredis"

	accumulatorCmd "github.com/rivalry-matchmaker/rivalry/cmd/accumulator/cmd"
	dispenserCmd "github.com/rivalry-matchmaker/rivalry/cmd/dispenser/cmd"
	frontendCmd "github.com/rivalry-matchmaker/rivalry/cmd/frontend/cmd"
	assignmentCmd "github.com/rivalry-matchmaker/rivalry/examples/assignment/cmd"
	matchmakerCmd "github.com/rivalry-matchmaker/rivalry/examples/matchmaker/cmd"
	"github.com/rivalry-matchmaker/rivalry/pkg/pb"
	"github.com/nats-io/nats.go"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var (
	frontendTarget          = "127.0.0.1:50051"
	matchmakerTarget        = "127.0.0.1:50052"
	assignmentServiceTarget = "127.0.0.1:50053"

	profile = &pb.MatchProfile{Name: "everybody", Pools: []*pb.Pool{{Name: "pool_everybody"}}}
)

func initRedis(s *suite.Suite) *tempredis.Server {
	server, err := tempredis.Start(tempredis.Config{})
	require.NoError(s.T(), err)

	return server
}

func initNats(s *suite.Suite) *server.Server {
	// Set Up NATS Server
	natsServer, err := server.NewServer(&server.Options{
		Host: "127.0.0.1",
		Port: nats.DefaultPort,
	})
	require.NoError(s.T(), err)
	err = server.Run(natsServer)
	require.NoError(s.T(), err)
	return natsServer
}

func initFrontend(s *suite.Suite, redisAddr, natsAddr string) *cobra.Command {
	profilesBytes, err := json.Marshal([]*pb.MatchProfile{profile})
	require.NoError(s.T(), err)
	frontendArgs := []string{
		"--redis_addr", redisAddr,
		"--redis_unix",
		"--nats_addr", natsAddr,
		"--address", frontendTarget,
		"--profiles", string(profilesBytes),
	}
	frontend := frontendCmd.NewRootCmd()
	frontend.SetArgs(frontendArgs)
	frontend.SilenceUsage = true
	frontend.SilenceErrors = true
	go frontend.Execute()
	return frontend
}

func stopFrontend() {
	frontendCmd.Stop()
}

func initAccumulator(s *suite.Suite, redisAddr, natsAddr string) *cobra.Command {
	profilesBytes, err := json.Marshal([]*pb.MatchProfile{profile})
	require.NoError(s.T(), err)
	accumulatorArgs := []string{
		"--redis_addr", redisAddr,
		"--redis_unix",
		"--nats_addr", natsAddr,
		"--profile", profile.Name,
		"--profiles", string(profilesBytes),
		"--matchmaker_target", matchmakerTarget,
		"--max_delay", fmt.Sprint(time.Second.Milliseconds() / 4),
	}
	accumulator := accumulatorCmd.NewRootCmd()
	accumulator.SetArgs(accumulatorArgs)
	accumulator.SilenceUsage = true
	accumulator.SilenceErrors = true
	go accumulator.Execute()
	return accumulator
}

func stopAccumulator() {
	accumulatorCmd.Stop()
}

func initDispenser(_ *suite.Suite, redisAddr, natsAddr string) *cobra.Command {
	dispenserArgs := []string{
		"--redis_addr", redisAddr,
		"--redis_unix",
		"--nats_addr", natsAddr,
		"--assignment_target", assignmentServiceTarget,
	}

	dispenser := dispenserCmd.NewRootCmd()
	dispenser.SetArgs(dispenserArgs)
	dispenser.SilenceUsage = true
	dispenser.SilenceErrors = true
	go dispenser.Execute()
	return dispenser
}

func stopDispenser() {
	dispenserCmd.Stop()
}

func initMatchmaker(backfill bool) *cobra.Command {
	matchmaker := matchmakerCmd.NewRootCmd()
	args := []string{
		"--address", matchmakerTarget,
	}
	if backfill {
		args = append(args, "--backfill")
	}
	matchmaker.SetArgs(args)
	matchmaker.SilenceUsage = true
	matchmaker.SilenceErrors = true
	go matchmaker.Execute()
	return matchmaker
}

func stopMatchmaker() {
	matchmakerCmd.Stop()
}

func initAssignmentService() *cobra.Command {
	assignmentService := assignmentCmd.NewRootCmd()
	assignmentService.SetArgs([]string{
		"--address", assignmentServiceTarget,
	})
	assignmentService.SilenceUsage = true
	assignmentService.SilenceErrors = true
	go assignmentService.Execute()
	return assignmentService
}

func stopAssignmentService() {
	assignmentCmd.Stop()
}

type player struct {
	playerID              string
	s                     suite.Suite
	wg                    *sync.WaitGroup
	frontendServiceClient pb.FrontendServiceClient
	assignment            *pb.Assignment
}

func newPlayer(s suite.Suite, wg *sync.WaitGroup, frontendServiceClient pb.FrontendServiceClient, playerID string) *player {
	return &player{
		s: s, wg: wg,
		playerID:              playerID,
		frontendServiceClient: frontendServiceClient,
	}
}

func (p *player) playTheGame(ctx context.Context) {
	log.Info().Msg("CreateTicket " + p.playerID)
	resp, err := p.frontendServiceClient.CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.NoError(p.s.T(), err)

	log.Info().Str("ticket", resp.Id).Msg("WatchAssignments Ticket " + p.playerID)
	cli, err := p.frontendServiceClient.WatchAssignments(ctx, &pb.WatchAssignmentsRequest{TicketId: resp.Id})
	require.NoError(p.s.T(), err)

	watchResp, err := cli.Recv()
	log.Info().Interface("assignment", watchResp).Msg("Assigned Ticket " + p.playerID)
	require.NoError(p.s.T(), err)
	require.NotNil(p.s.T(), watchResp.Assignment)
	assert.NotEmpty(p.s.T(), watchResp.Assignment.Connection)
	p.assignment = watchResp.Assignment
	p.wg.Done()
}
