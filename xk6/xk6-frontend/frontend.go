package frontend

import (
	"fmt"
	"sync"

	"github.com/rivalry-matchmaker/rivalry/pkg/pb"
	"go.k6.io/k6/js/modules"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func init() {
	modules.Register("k6/x/frontend", new(RootModule))
}

// Ensure the interfaces are implemented correctly.
var (
	_ modules.Module   = &RootModule{}
	_ modules.Instance = &Client{}
)

type RootModule struct{}

// NewModuleInstance implements the modules.Module interface to return
// a new instance for each VU.
func (*RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	return NewClient(vu)
}

type Client struct {
	vu     modules.VU
	cli    pb.FrontendServiceClient
	cliMux sync.Mutex
}

func NewClient(vu modules.VU) *Client {
	return &Client{
		vu: vu,
	}
}

// Exports implements the modules.Instance interface and returns the exports
// of the JS module.
func (c *Client) Exports() modules.Exports {
	return modules.Exports{Default: c}
}

func (c *Client) getCli(target string) (pb.FrontendServiceClient, error) {
	c.cliMux.Lock()
	defer c.cliMux.Unlock()
	if c.cli != nil {
		return c.cli, nil
	}
	conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC: %w", err)
	}
	c.cli = pb.NewFrontendServiceClient(conn)
	return c.cli, nil
}

func (c *Client) MatchRequest(target string, ticket *pb.Ticket) (string, error) {
	cli, err := c.getCli(target)
	if err != nil {
		return "", err
	}

	if resp, err := cli.CreateTicket(c.vu.Context(), &pb.CreateTicketRequest{
		Ticket: ticket}); err != nil {
		return "", err
	} else {
		return resp.Id, nil
	}
}

func (c *Client) MatchStatus(target string, ticketID string) (*pb.Assignment, error) {
	cli, err := c.getCli(target)
	if err != nil {
		return nil, err
	}

	stream, err := cli.WatchAssignments(c.vu.Context(), &pb.WatchAssignmentsRequest{TicketId: ticketID})
	if err != nil {
		return nil, err
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			return nil, err
		}
		if resp.Assignment != nil {
			return resp.Assignment, nil
		}
	}
}
