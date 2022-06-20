// Code generated by MockGen. DO NOT EDIT.
// Source: tpl.go

// Package render is a generated GoMock package.
package render

import (
	reflect "reflect"
	template "text/template"

	gomock "github.com/golang/mock/gomock"
)

// MockTplProvider is a mock of TplProvider interface.
type MockTplProvider struct {
	ctrl     *gomock.Controller
	recorder *MockTplProviderMockRecorder
}

// MockTplProviderMockRecorder is the mock recorder for MockTplProvider.
type MockTplProviderMockRecorder struct {
	mock *MockTplProvider
}

// NewMockTplProvider creates a new mock instance.
func NewMockTplProvider(ctrl *gomock.Controller) *MockTplProvider {
	mock := &MockTplProvider{ctrl: ctrl}
	mock.recorder = &MockTplProviderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTplProvider) EXPECT() *MockTplProviderMockRecorder {
	return m.recorder
}

// GetTpl mocks base method.
func (m *MockTplProvider) GetTpl(tplText string) (*template.Template, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTpl", tplText)
	ret0, _ := ret[0].(*template.Template)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTpl indicates an expected call of GetTpl.
func (mr *MockTplProviderMockRecorder) GetTpl(tplText interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTpl", reflect.TypeOf((*MockTplProvider)(nil).GetTpl), tplText)
}
