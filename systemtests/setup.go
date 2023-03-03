package systemtests

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	matcherCmd "github.com/rivalry-matchmaker/rivalry/cmd/matcher/cmd"
	api "github.com/rivalry-matchmaker/rivalry/pkg/pb/api/v1"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stvp/tempredis"

	"github.com/nats-io/nats.go"
	accumulatorCmd "github.com/rivalry-matchmaker/rivalry/cmd/accumulator/cmd"
	dispenserCmd "github.com/rivalry-matchmaker/rivalry/cmd/dispenser/cmd"
	frontendCmd "github.com/rivalry-matchmaker/rivalry/cmd/frontend/cmd"
	assignmentCmd "github.com/rivalry-matchmaker/rivalry/examples/assignment/cmd"
	matchmakerCmd "github.com/rivalry-matchmaker/rivalry/examples/matchmaker/cmd"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var (
	frontendTarget          = "127.0.0.1:50051"
	matchmakerTarget        = "127.0.0.1:50052"
	assignmentServiceTarget = "127.0.0.1:50053"
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
	frontendArgs := []string{
		"--redis_addr", redisAddr,
		"--redis_unix",
		"--nats_addr", natsAddr,
		"--address", frontendTarget,
	}
	frontend := frontendCmd.NewRootCmd()
	frontend.SetArgs(frontendArgs)
	frontend.SilenceUsage = true
	frontend.SilenceErrors = true
	go func() {
		err := frontend.Execute()
		if err != nil {
			log.Err(err).Msg("failed to execute frontend")
		}
	}()
	return frontend
}

func stopFrontend() {
	frontendCmd.Stop()
}

func initAccumulator(s *suite.Suite, redisAddr, natsAddr string) *cobra.Command {
	accumulatorArgs := []string{
		"--redis_addr", redisAddr,
		"--redis_unix",
		"--nats_addr", natsAddr,
		"--max_delay", fmt.Sprint(time.Second.Milliseconds() / 4),
	}
	accumulator := accumulatorCmd.NewRootCmd()
	accumulator.SetArgs(accumulatorArgs)
	accumulator.SilenceUsage = true
	accumulator.SilenceErrors = true
	go func() {
		err := accumulator.Execute()
		if err != nil {
			log.Err(err).Msg("failed to execute accumulator")
		}
	}()
	return accumulator
}

func stopAccumulator() {
	accumulatorCmd.Stop()
}

func initMatcher(s *suite.Suite, redisAddr, natsAddr string) *cobra.Command {
	matcherArgs := []string{
		"--redis_addr", redisAddr,
		"--redis_unix",
		"--nats_addr", natsAddr,
		"--matchmaker_target", matchmakerTarget,
	}
	matcher := matcherCmd.NewRootCmd()
	matcher.SetArgs(matcherArgs)
	matcher.SilenceUsage = true
	matcher.SilenceErrors = true
	go func() {
		err := matcher.Execute()
		if err != nil {
			log.Err(err).Msg("failed to execute matcher")
		}
	}()
	return matcher
}

func stopMatcher() {
	matchmakerCmd.Stop()
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
	go func() {
		err := dispenser.Execute()
		if err != nil {
			log.Err(err).Msg("failed to execute dispenser")
		}
	}()
	return dispenser
}

func stopDispenser() {
	dispenserCmd.Stop()
}

func initMatchmaker() *cobra.Command {
	matchmaker := matchmakerCmd.NewRootCmd()
	args := []string{
		"--address", matchmakerTarget,
	}
	matchmaker.SetArgs(args)
	matchmaker.SilenceUsage = true
	matchmaker.SilenceErrors = true
	go func() {
		err := matchmaker.Execute()
		if err != nil {
			log.Err(err).Msg("failed to execute matchmaker")
		}
	}()
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
	go func() {
		err := assignmentService.Execute()
		if err != nil {
			log.Err(err).Msg("failed to execute assignment service")
		}
	}()
	return assignmentService
}

func stopAssignmentService() {
	assignmentCmd.Stop()
}

type player struct {
	playerID              string
	s                     suite.Suite
	wg                    *sync.WaitGroup
	frontendServiceClient api.RivalryServiceClient
	assignment            *api.GameServer
}

func newPlayer(s suite.Suite, wg *sync.WaitGroup, frontendServiceClient api.RivalryServiceClient, playerID string) *player {
	return &player{
		s: s, wg: wg,
		playerID:              playerID,
		frontendServiceClient: frontendServiceClient,
	}
}

func (p *player) playTheGame(ctx context.Context) {
	log.Info().Msg("CreateMatchRequest " + p.playerID)
	cli, err := p.frontendServiceClient.Match(ctx, &api.MatchRequest{
		PlayerId:         p.playerID,
		MatchmakingQueue: "default",
	})
	require.NoError(p.s.T(), err)

	watchResp, err := cli.Recv()
	log.Info().Interface("assignment", watchResp).Msg("Assigned Ticket " + p.playerID)
	require.NoError(p.s.T(), err)
	require.NotNil(p.s.T(), watchResp.GameServer)
	assert.NotEmpty(p.s.T(), watchResp.GameServer.GameServerIp)
	p.assignment = watchResp.GameServer
	p.wg.Done()
}
