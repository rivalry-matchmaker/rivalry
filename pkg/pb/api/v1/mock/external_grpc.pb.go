// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/rivalry-matchmaker/rivalry/pkg/pb/api/v1 (interfaces: RivalryServiceClient,RivalryService_MatchClient,RivalryServiceServer,UnsafeRivalryServiceServer,RivalryService_MatchServer)

// Package mock_pb is a generated GoMock package.
package mock_pb

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	v1 "github.com/rivalry-matchmaker/rivalry/pkg/pb/api/v1"
	grpc "google.golang.org/grpc"
	metadata "google.golang.org/grpc/metadata"
)

// MockRivalryServiceClient is a mock of RivalryServiceClient interface.
type MockRivalryServiceClient struct {
	ctrl     *gomock.Controller
	recorder *MockRivalryServiceClientMockRecorder
}

// MockRivalryServiceClientMockRecorder is the mock recorder for MockRivalryServiceClient.
type MockRivalryServiceClientMockRecorder struct {
	mock *MockRivalryServiceClient
}

// NewMockRivalryServiceClient creates a new mock instance.
func NewMockRivalryServiceClient(ctrl *gomock.Controller) *MockRivalryServiceClient {
	mock := &MockRivalryServiceClient{ctrl: ctrl}
	mock.recorder = &MockRivalryServiceClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRivalryServiceClient) EXPECT() *MockRivalryServiceClientMockRecorder {
	return m.recorder
}

// Match mocks base method.
func (m *MockRivalryServiceClient) Match(arg0 context.Context, arg1 *v1.MatchRequest, arg2 ...grpc.CallOption) (v1.RivalryService_MatchClient, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Match", varargs...)
	ret0, _ := ret[0].(v1.RivalryService_MatchClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Match indicates an expected call of Match.
func (mr *MockRivalryServiceClientMockRecorder) Match(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Match", reflect.TypeOf((*MockRivalryServiceClient)(nil).Match), varargs...)
}

// MockRivalryService_MatchClient is a mock of RivalryService_MatchClient interface.
type MockRivalryService_MatchClient struct {
	ctrl     *gomock.Controller
	recorder *MockRivalryService_MatchClientMockRecorder
}

// MockRivalryService_MatchClientMockRecorder is the mock recorder for MockRivalryService_MatchClient.
type MockRivalryService_MatchClientMockRecorder struct {
	mock *MockRivalryService_MatchClient
}

// NewMockRivalryService_MatchClient creates a new mock instance.
func NewMockRivalryService_MatchClient(ctrl *gomock.Controller) *MockRivalryService_MatchClient {
	mock := &MockRivalryService_MatchClient{ctrl: ctrl}
	mock.recorder = &MockRivalryService_MatchClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRivalryService_MatchClient) EXPECT() *MockRivalryService_MatchClientMockRecorder {
	return m.recorder
}

// CloseSend mocks base method.
func (m *MockRivalryService_MatchClient) CloseSend() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CloseSend")
	ret0, _ := ret[0].(error)
	return ret0
}

// CloseSend indicates an expected call of CloseSend.
func (mr *MockRivalryService_MatchClientMockRecorder) CloseSend() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CloseSend", reflect.TypeOf((*MockRivalryService_MatchClient)(nil).CloseSend))
}

// Context mocks base method.
func (m *MockRivalryService_MatchClient) Context() context.Context {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Context")
	ret0, _ := ret[0].(context.Context)
	return ret0
}

// Context indicates an expected call of Context.
func (mr *MockRivalryService_MatchClientMockRecorder) Context() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Context", reflect.TypeOf((*MockRivalryService_MatchClient)(nil).Context))
}

// Header mocks base method.
func (m *MockRivalryService_MatchClient) Header() (metadata.MD, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Header")
	ret0, _ := ret[0].(metadata.MD)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Header indicates an expected call of Header.
func (mr *MockRivalryService_MatchClientMockRecorder) Header() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Header", reflect.TypeOf((*MockRivalryService_MatchClient)(nil).Header))
}

// Recv mocks base method.
func (m *MockRivalryService_MatchClient) Recv() (*v1.MatchResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Recv")
	ret0, _ := ret[0].(*v1.MatchResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Recv indicates an expected call of Recv.
func (mr *MockRivalryService_MatchClientMockRecorder) Recv() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Recv", reflect.TypeOf((*MockRivalryService_MatchClient)(nil).Recv))
}

// RecvMsg mocks base method.
func (m *MockRivalryService_MatchClient) RecvMsg(arg0 interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RecvMsg", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// RecvMsg indicates an expected call of RecvMsg.
func (mr *MockRivalryService_MatchClientMockRecorder) RecvMsg(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecvMsg", reflect.TypeOf((*MockRivalryService_MatchClient)(nil).RecvMsg), arg0)
}

// SendMsg mocks base method.
func (m *MockRivalryService_MatchClient) SendMsg(arg0 interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendMsg", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMsg indicates an expected call of SendMsg.
func (mr *MockRivalryService_MatchClientMockRecorder) SendMsg(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMsg", reflect.TypeOf((*MockRivalryService_MatchClient)(nil).SendMsg), arg0)
}

// Trailer mocks base method.
func (m *MockRivalryService_MatchClient) Trailer() metadata.MD {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Trailer")
	ret0, _ := ret[0].(metadata.MD)
	return ret0
}

// Trailer indicates an expected call of Trailer.
func (mr *MockRivalryService_MatchClientMockRecorder) Trailer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Trailer", reflect.TypeOf((*MockRivalryService_MatchClient)(nil).Trailer))
}

// MockRivalryServiceServer is a mock of RivalryServiceServer interface.
type MockRivalryServiceServer struct {
	ctrl     *gomock.Controller
	recorder *MockRivalryServiceServerMockRecorder
}

// MockRivalryServiceServerMockRecorder is the mock recorder for MockRivalryServiceServer.
type MockRivalryServiceServerMockRecorder struct {
	mock *MockRivalryServiceServer
}

// NewMockRivalryServiceServer creates a new mock instance.
func NewMockRivalryServiceServer(ctrl *gomock.Controller) *MockRivalryServiceServer {
	mock := &MockRivalryServiceServer{ctrl: ctrl}
	mock.recorder = &MockRivalryServiceServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRivalryServiceServer) EXPECT() *MockRivalryServiceServerMockRecorder {
	return m.recorder
}

// Match mocks base method.
func (m *MockRivalryServiceServer) Match(arg0 *v1.MatchRequest, arg1 v1.RivalryService_MatchServer) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Match", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Match indicates an expected call of Match.
func (mr *MockRivalryServiceServerMockRecorder) Match(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Match", reflect.TypeOf((*MockRivalryServiceServer)(nil).Match), arg0, arg1)
}

// MockUnsafeRivalryServiceServer is a mock of UnsafeRivalryServiceServer interface.
type MockUnsafeRivalryServiceServer struct {
	ctrl     *gomock.Controller
	recorder *MockUnsafeRivalryServiceServerMockRecorder
}

// MockUnsafeRivalryServiceServerMockRecorder is the mock recorder for MockUnsafeRivalryServiceServer.
type MockUnsafeRivalryServiceServerMockRecorder struct {
	mock *MockUnsafeRivalryServiceServer
}

// NewMockUnsafeRivalryServiceServer creates a new mock instance.
func NewMockUnsafeRivalryServiceServer(ctrl *gomock.Controller) *MockUnsafeRivalryServiceServer {
	mock := &MockUnsafeRivalryServiceServer{ctrl: ctrl}
	mock.recorder = &MockUnsafeRivalryServiceServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUnsafeRivalryServiceServer) EXPECT() *MockUnsafeRivalryServiceServerMockRecorder {
	return m.recorder
}

// mustEmbedUnimplementedRivalryServiceServer mocks base method.
func (m *MockUnsafeRivalryServiceServer) mustEmbedUnimplementedRivalryServiceServer() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "mustEmbedUnimplementedRivalryServiceServer")
}

// mustEmbedUnimplementedRivalryServiceServer indicates an expected call of mustEmbedUnimplementedRivalryServiceServer.
func (mr *MockUnsafeRivalryServiceServerMockRecorder) mustEmbedUnimplementedRivalryServiceServer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "mustEmbedUnimplementedRivalryServiceServer", reflect.TypeOf((*MockUnsafeRivalryServiceServer)(nil).mustEmbedUnimplementedRivalryServiceServer))
}

// MockRivalryService_MatchServer is a mock of RivalryService_MatchServer interface.
type MockRivalryService_MatchServer struct {
	ctrl     *gomock.Controller
	recorder *MockRivalryService_MatchServerMockRecorder
}

// MockRivalryService_MatchServerMockRecorder is the mock recorder for MockRivalryService_MatchServer.
type MockRivalryService_MatchServerMockRecorder struct {
	mock *MockRivalryService_MatchServer
}

// NewMockRivalryService_MatchServer creates a new mock instance.
func NewMockRivalryService_MatchServer(ctrl *gomock.Controller) *MockRivalryService_MatchServer {
	mock := &MockRivalryService_MatchServer{ctrl: ctrl}
	mock.recorder = &MockRivalryService_MatchServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRivalryService_MatchServer) EXPECT() *MockRivalryService_MatchServerMockRecorder {
	return m.recorder
}

// Context mocks base method.
func (m *MockRivalryService_MatchServer) Context() context.Context {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Context")
	ret0, _ := ret[0].(context.Context)
	return ret0
}

// Context indicates an expected call of Context.
func (mr *MockRivalryService_MatchServerMockRecorder) Context() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Context", reflect.TypeOf((*MockRivalryService_MatchServer)(nil).Context))
}

// RecvMsg mocks base method.
func (m *MockRivalryService_MatchServer) RecvMsg(arg0 interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RecvMsg", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// RecvMsg indicates an expected call of RecvMsg.
func (mr *MockRivalryService_MatchServerMockRecorder) RecvMsg(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecvMsg", reflect.TypeOf((*MockRivalryService_MatchServer)(nil).RecvMsg), arg0)
}

// Send mocks base method.
func (m *MockRivalryService_MatchServer) Send(arg0 *v1.MatchResponse) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Send", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Send indicates an expected call of Send.
func (mr *MockRivalryService_MatchServerMockRecorder) Send(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Send", reflect.TypeOf((*MockRivalryService_MatchServer)(nil).Send), arg0)
}

// SendHeader mocks base method.
func (m *MockRivalryService_MatchServer) SendHeader(arg0 metadata.MD) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendHeader", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendHeader indicates an expected call of SendHeader.
func (mr *MockRivalryService_MatchServerMockRecorder) SendHeader(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendHeader", reflect.TypeOf((*MockRivalryService_MatchServer)(nil).SendHeader), arg0)
}

// SendMsg mocks base method.
func (m *MockRivalryService_MatchServer) SendMsg(arg0 interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendMsg", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMsg indicates an expected call of SendMsg.
func (mr *MockRivalryService_MatchServerMockRecorder) SendMsg(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMsg", reflect.TypeOf((*MockRivalryService_MatchServer)(nil).SendMsg), arg0)
}

// SetHeader mocks base method.
func (m *MockRivalryService_MatchServer) SetHeader(arg0 metadata.MD) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetHeader", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetHeader indicates an expected call of SetHeader.
func (mr *MockRivalryService_MatchServerMockRecorder) SetHeader(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetHeader", reflect.TypeOf((*MockRivalryService_MatchServer)(nil).SetHeader), arg0)
}

// SetTrailer mocks base method.
func (m *MockRivalryService_MatchServer) SetTrailer(arg0 metadata.MD) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetTrailer", arg0)
}

// SetTrailer indicates an expected call of SetTrailer.
func (mr *MockRivalryService_MatchServerMockRecorder) SetTrailer(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetTrailer", reflect.TypeOf((*MockRivalryService_MatchServer)(nil).SetTrailer), arg0)
}