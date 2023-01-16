package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/rivalry-matchmaker/rivalry/pkg/pb"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

// FakeServerAssignmentService conforms to the AssignmentServiceServer interface
type FakeServerAssignmentService struct {
	pb.UnimplementedAssignmentServiceServer
	ctx            context.Context
	frontendClient pb.FrontendServiceClient
}

// NewFakeServerAssignmentService returns a AssignmentServiceServer that makes fake assignments
func NewFakeServerAssignmentService(ctx context.Context, frontendTarget string) pb.AssignmentServiceServer {
	conn, err := grpc.Dial(frontendTarget, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	return NewFakeServerAssignmentServiceWithClient(ctx, pb.NewFrontendServiceClient(conn))
}

// NewFakeServerAssignmentServiceWithClient returns a AssignmentServiceServer that makes fake assignments
func NewFakeServerAssignmentServiceWithClient(ctx context.Context, frontendClient pb.FrontendServiceClient) pb.AssignmentServiceServer {
	return &FakeServerAssignmentService{
		ctx:            ctx,
		frontendClient: frontendClient,
	}
}

// MakeAssignment returns a fake assignment
func (s *FakeServerAssignmentService) MakeAssignment(_ context.Context, match *pb.Match) (*pb.Assignment, error) {
	// generate a connection string with a random port number
	connection := "127.0.0.1:" + fmt.Sprint(1024+rand.Intn(65535-1024))

	// If the match contains a backfill then we simulate the role of the game server and
	// acknowledge the backfill.
	if match.Backfill != nil {
		go func() {
			for range time.Tick(time.Second) {
				select {
				case <-s.ctx.Done():
					return
				default:
					_, err := s.frontendClient.AcknowledgeBackfill(context.Background(), &pb.AcknowledgeBackfillRequest{
						BackfillId: match.Backfill.Id, Assignment: &pb.Assignment{Connection: connection}})
					if err != nil {
						log.Err(err).Msg("failed to acknowledge backfill")
					}
				}
			}
		}()
	}

	// return assignment
	return &pb.Assignment{Connection: connection}, nil
}
