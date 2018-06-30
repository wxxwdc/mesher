package health

import (
	"github.com/stretchr/testify/mock"
)

type MockMemberDiscovery struct {
	mock.Mock
}

func (m *MockMemberDiscovery) ConfigurationInit(initConfigServer []string) error {
	args := m.Called(initConfigServer)
	return args.Error(0)
}
func (m *MockMemberDiscovery) GetConfigServer() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}
func (m *MockMemberDiscovery) RefreshMembers() error {
	args := m.Called()
	return args.Error(0)
}
func (m *MockMemberDiscovery) Shuffle() error {
	args := m.Called()
	return args.Error(0)
}
func (m *MockMemberDiscovery) GetWorkingConfigCenterIP(entryPoint []string) ([]string, error) {
	args := m.Called(entryPoint)
	return args.Get(0).([]string), args.Error(0)
}
