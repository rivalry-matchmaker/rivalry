// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/rivalry-matchmaker/rivalry/internal/managers/tickets (interfaces: Manager)

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	v1 "github.com/rivalry-matchmaker/rivalry/pkg/pb/api/v1"
	v10 "github.com/rivalry-matchmaker/rivalry/pkg/pb/db/v1"
	v11 "github.com/rivalry-matchmaker/rivalry/pkg/pb/stream/v1"
)

// MockManager is a mock of Manager interface.
type MockManager struct {
	ctrl     *gomock.Controller
	recorder *MockManagerMockRecorder
}

// MockManagerMockRecorder is the mock recorder for MockManager.
type MockManagerMockRecorder struct {
	mock *MockManager
}

// NewMockManager creates a new mock instance.
func NewMockManager(ctrl *gomock.Controller) *MockManager {
	mock := &MockManager{ctrl: ctrl}
	mock.recorder = &MockManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockManager) EXPECT() *MockManagerMockRecorder {
	return m.recorder
}

// AddAssignmentToMatchRequests mocks base method.
func (m *MockManager) AddAssignmentToMatchRequests(arg0 context.Context, arg1 *v1.GameServer, arg2 *v1.Match) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddAssignmentToMatchRequests", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddAssignmentToMatchRequests indicates an expected call of AddAssignmentToMatchRequests.
func (mr *MockManagerMockRecorder) AddAssignmentToMatchRequests(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddAssignmentToMatchRequests", reflect.TypeOf((*MockManager)(nil).AddAssignmentToMatchRequests), arg0, arg1, arg2)
}

// AssignMatchRequestsToMatch mocks base method.
func (m *MockManager) AssignMatchRequestsToMatch(arg0 context.Context, arg1 *v1.Match) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AssignMatchRequestsToMatch", arg0, arg1)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AssignMatchRequestsToMatch indicates an expected call of AssignMatchRequestsToMatch.
func (mr *MockManagerMockRecorder) AssignMatchRequestsToMatch(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AssignMatchRequestsToMatch", reflect.TypeOf((*MockManager)(nil).AssignMatchRequestsToMatch), arg0, arg1)
}

// Close mocks base method.
func (m *MockManager) Close() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Close")
}

// Close indicates an expected call of Close.
func (mr *MockManagerMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockManager)(nil).Close))
}

// CreateMatchRequest mocks base method.
func (m *MockManager) CreateMatchRequest(arg0 context.Context, arg1 *v1.MatchRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateMatchRequest", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateMatchRequest indicates an expected call of CreateMatchRequest.
func (mr *MockManagerMockRecorder) CreateMatchRequest(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateMatchRequest", reflect.TypeOf((*MockManager)(nil).CreateMatchRequest), arg0, arg1)
}

// DeleteMatchRequest mocks base method.
func (m *MockManager) DeleteMatchRequest(arg0 context.Context, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteMatchRequest", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteMatchRequest indicates an expected call of DeleteMatchRequest.
func (mr *MockManagerMockRecorder) DeleteMatchRequest(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteMatchRequest", reflect.TypeOf((*MockManager)(nil).DeleteMatchRequest), arg0, arg1)
}

// GetMatchRequest mocks base method.
func (m *MockManager) GetMatchRequest(arg0 context.Context, arg1 string) (*v10.MatchRequest, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMatchRequest", arg0, arg1)
	ret0, _ := ret[0].(*v10.MatchRequest)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMatchRequest indicates an expected call of GetMatchRequest.
func (mr *MockManagerMockRecorder) GetMatchRequest(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMatchRequest", reflect.TypeOf((*MockManager)(nil).GetMatchRequest), arg0, arg1)
}

// GetMatchRequests mocks base method.
func (m *MockManager) GetMatchRequests(arg0 context.Context, arg1 []string) ([]*v10.MatchRequest, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMatchRequests", arg0, arg1)
	ret0, _ := ret[0].([]*v10.MatchRequest)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMatchRequests indicates an expected call of GetMatchRequests.
func (mr *MockManagerMockRecorder) GetMatchRequests(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMatchRequests", reflect.TypeOf((*MockManager)(nil).GetMatchRequests), arg0, arg1)
}

// PublishAccumulatedMatchRequests mocks base method.
func (m *MockManager) PublishAccumulatedMatchRequests(arg0 context.Context, arg1 string, arg2 []*v11.StreamTicket) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PublishAccumulatedMatchRequests", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// PublishAccumulatedMatchRequests indicates an expected call of PublishAccumulatedMatchRequests.
func (mr *MockManagerMockRecorder) PublishAccumulatedMatchRequests(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PublishAccumulatedMatchRequests", reflect.TypeOf((*MockManager)(nil).PublishAccumulatedMatchRequests), arg0, arg1, arg2)
}

// ReleaseMatchRequestsFromMatch mocks base method.
func (m *MockManager) ReleaseMatchRequestsFromMatch(arg0 context.Context, arg1 *v1.Match) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReleaseMatchRequestsFromMatch", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReleaseMatchRequestsFromMatch indicates an expected call of ReleaseMatchRequestsFromMatch.
func (mr *MockManagerMockRecorder) ReleaseMatchRequestsFromMatch(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReleaseMatchRequestsFromMatch", reflect.TypeOf((*MockManager)(nil).ReleaseMatchRequestsFromMatch), arg0, arg1)
}

// RequeueMatchRequests mocks base method.
func (m *MockManager) RequeueMatchRequests(arg0 context.Context, arg1 []*v11.StreamTicket) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "RequeueMatchRequests", arg0, arg1)
}

// RequeueMatchRequests indicates an expected call of RequeueMatchRequests.
func (mr *MockManagerMockRecorder) RequeueMatchRequests(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RequeueMatchRequests", reflect.TypeOf((*MockManager)(nil).RequeueMatchRequests), arg0, arg1)
}

// StreamAccumulatedMatchRequests mocks base method.
func (m *MockManager) StreamAccumulatedMatchRequests(arg0 context.Context, arg1 string, arg2 func(context.Context, []*v11.StreamTicket)) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StreamAccumulatedMatchRequests", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// StreamAccumulatedMatchRequests indicates an expected call of StreamAccumulatedMatchRequests.
func (mr *MockManagerMockRecorder) StreamAccumulatedMatchRequests(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StreamAccumulatedMatchRequests", reflect.TypeOf((*MockManager)(nil).StreamAccumulatedMatchRequests), arg0, arg1, arg2)
}

// StreamMatchRequests mocks base method.
func (m *MockManager) StreamMatchRequests(arg0 context.Context, arg1 string, arg2 func(context.Context, *v11.StreamTicket)) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StreamMatchRequests", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// StreamMatchRequests indicates an expected call of StreamMatchRequests.
func (mr *MockManagerMockRecorder) StreamMatchRequests(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StreamMatchRequests", reflect.TypeOf((*MockManager)(nil).StreamMatchRequests), arg0, arg1, arg2)
}

// WatchMatchRequest mocks base method.
func (m *MockManager) WatchMatchRequest(arg0 context.Context, arg1 string, arg2 func(context.Context, *v10.MatchRequest)) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchMatchRequest", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// WatchMatchRequest indicates an expected call of WatchMatchRequest.
func (mr *MockManagerMockRecorder) WatchMatchRequest(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchMatchRequest", reflect.TypeOf((*MockManager)(nil).WatchMatchRequest), arg0, arg1, arg2)
}
