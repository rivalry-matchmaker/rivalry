package service

import (
	"context"
	"math/rand"

	api "github.com/rivalry-matchmaker/rivalry/pkg/pb/api/v1"
	"google.golang.org/grpc"
)

// FakeServerAssignmentService conforms to the AssignmentServiceServer interface
type FakeServerAssignmentService struct {
	api.UnimplementedAssignmentServiceServer
	frontendClient api.RivalryServiceClient
}

// NewFakeServerAssignmentService returns a AssignmentServiceServer that makes fake assignments
func NewFakeServerAssignmentService(frontendTarget string) api.AssignmentServiceServer {
	conn, err := grpc.Dial(frontendTarget, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	return NewFakeServerAssignmentServiceWithClient(api.NewRivalryServiceClient(conn))
}

// NewFakeServerAssignmentServiceWithClient returns a AssignmentServiceServer that makes fake assignments
func NewFakeServerAssignmentServiceWithClient(frontendClient api.RivalryServiceClient) api.AssignmentServiceServer {
	return &FakeServerAssignmentService{
		frontendClient: frontendClient,
	}
}

// MakeAssignment returns a fake assignment
func (s *FakeServerAssignmentService) MakeAssignment(context.Context, *api.MakeAssignmentRequest) (*api.MakeAssignmentResponse, error) {
	// return assignment
	return &api.MakeAssignmentResponse{
		GameServer: &api.GameServer{
			GameServerIp:    "127.0.0.1",
			GameServerPorts: []*api.GameServerPort{{Name: "game", Port: int32(1024 + rand.Intn(65535-1024))}},
		},
	}, nil
}
