// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             (unknown)
// source: api/v1/match_logic.proto

package v1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// MatchMakerServiceClient is the client API for MatchMakerService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type MatchMakerServiceClient interface {
	// MakeMatches takes a MatchProfile and a map of pool names to ticket slices, and creates
	// a slice of Match's from that information
	MakeMatches(ctx context.Context, in *MakeMatchesRequest, opts ...grpc.CallOption) (MatchMakerService_MakeMatchesClient, error)
}

type matchMakerServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewMatchMakerServiceClient(cc grpc.ClientConnInterface) MatchMakerServiceClient {
	return &matchMakerServiceClient{cc}
}

func (c *matchMakerServiceClient) MakeMatches(ctx context.Context, in *MakeMatchesRequest, opts ...grpc.CallOption) (MatchMakerService_MakeMatchesClient, error) {
	stream, err := c.cc.NewStream(ctx, &MatchMakerService_ServiceDesc.Streams[0], "/api.v1.MatchMakerService/MakeMatches", opts...)
	if err != nil {
		return nil, err
	}
	x := &matchMakerServiceMakeMatchesClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type MatchMakerService_MakeMatchesClient interface {
	Recv() (*MakeMatchesResponse, error)
	grpc.ClientStream
}

type matchMakerServiceMakeMatchesClient struct {
	grpc.ClientStream
}

func (x *matchMakerServiceMakeMatchesClient) Recv() (*MakeMatchesResponse, error) {
	m := new(MakeMatchesResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// MatchMakerServiceServer is the server API for MatchMakerService service.
// All implementations should embed UnimplementedMatchMakerServiceServer
// for forward compatibility
type MatchMakerServiceServer interface {
	// MakeMatches takes a MatchProfile and a map of pool names to ticket slices, and creates
	// a slice of Match's from that information
	MakeMatches(*MakeMatchesRequest, MatchMakerService_MakeMatchesServer) error
}

// UnimplementedMatchMakerServiceServer should be embedded to have forward compatible implementations.
type UnimplementedMatchMakerServiceServer struct {
}

func (UnimplementedMatchMakerServiceServer) MakeMatches(*MakeMatchesRequest, MatchMakerService_MakeMatchesServer) error {
	return status.Errorf(codes.Unimplemented, "method MakeMatches not implemented")
}

// UnsafeMatchMakerServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to MatchMakerServiceServer will
// result in compilation errors.
type UnsafeMatchMakerServiceServer interface {
	mustEmbedUnimplementedMatchMakerServiceServer()
}

func RegisterMatchMakerServiceServer(s grpc.ServiceRegistrar, srv MatchMakerServiceServer) {
	s.RegisterService(&MatchMakerService_ServiceDesc, srv)
}

func _MatchMakerService_MakeMatches_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(MakeMatchesRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(MatchMakerServiceServer).MakeMatches(m, &matchMakerServiceMakeMatchesServer{stream})
}

type MatchMakerService_MakeMatchesServer interface {
	Send(*MakeMatchesResponse) error
	grpc.ServerStream
}

type matchMakerServiceMakeMatchesServer struct {
	grpc.ServerStream
}

func (x *matchMakerServiceMakeMatchesServer) Send(m *MakeMatchesResponse) error {
	return x.ServerStream.SendMsg(m)
}

// MatchMakerService_ServiceDesc is the grpc.ServiceDesc for MatchMakerService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var MatchMakerService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "api.v1.MatchMakerService",
	HandlerType: (*MatchMakerServiceServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "MakeMatches",
			Handler:       _MatchMakerService_MakeMatches_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "api/v1/match_logic.proto",
}

// AssignmentServiceClient is the client API for AssignmentService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type AssignmentServiceClient interface {
	MakeAssignment(ctx context.Context, in *MakeAssignmentRequest, opts ...grpc.CallOption) (*MakeAssignmentResponse, error)
}

type assignmentServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewAssignmentServiceClient(cc grpc.ClientConnInterface) AssignmentServiceClient {
	return &assignmentServiceClient{cc}
}

func (c *assignmentServiceClient) MakeAssignment(ctx context.Context, in *MakeAssignmentRequest, opts ...grpc.CallOption) (*MakeAssignmentResponse, error) {
	out := new(MakeAssignmentResponse)
	err := c.cc.Invoke(ctx, "/api.v1.AssignmentService/MakeAssignment", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AssignmentServiceServer is the server API for AssignmentService service.
// All implementations should embed UnimplementedAssignmentServiceServer
// for forward compatibility
type AssignmentServiceServer interface {
	MakeAssignment(context.Context, *MakeAssignmentRequest) (*MakeAssignmentResponse, error)
}

// UnimplementedAssignmentServiceServer should be embedded to have forward compatible implementations.
type UnimplementedAssignmentServiceServer struct {
}

func (UnimplementedAssignmentServiceServer) MakeAssignment(context.Context, *MakeAssignmentRequest) (*MakeAssignmentResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MakeAssignment not implemented")
}

// UnsafeAssignmentServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to AssignmentServiceServer will
// result in compilation errors.
type UnsafeAssignmentServiceServer interface {
	mustEmbedUnimplementedAssignmentServiceServer()
}

func RegisterAssignmentServiceServer(s grpc.ServiceRegistrar, srv AssignmentServiceServer) {
	s.RegisterService(&AssignmentService_ServiceDesc, srv)
}

func _AssignmentService_MakeAssignment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MakeAssignmentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AssignmentServiceServer).MakeAssignment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.v1.AssignmentService/MakeAssignment",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AssignmentServiceServer).MakeAssignment(ctx, req.(*MakeAssignmentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// AssignmentService_ServiceDesc is the grpc.ServiceDesc for AssignmentService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var AssignmentService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "api.v1.AssignmentService",
	HandlerType: (*AssignmentServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "MakeAssignment",
			Handler:    _AssignmentService_MakeAssignment_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api/v1/match_logic.proto",
}