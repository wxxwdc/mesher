package mock

import (
	"github.com/ServiceComb/go-chassis/core/config/model"
	"github.com/stretchr/testify/mock"
)

type RouterMock struct {
	mock.Mock
}

func (m *RouterMock) Init() error {
	return nil
}
func (m *RouterMock) SetRouteRule(map[string][]*model.RouteRule) {}

func (m *RouterMock) FetchRouteRule() map[string][]*model.RouteRule {
	return nil
}

func (m *RouterMock) FetchRouteRuleByServiceName(s string) []*model.RouteRule {
	args := m.Called(s)
	rules, ok := args.Get(0).([]*model.RouteRule)
	if !ok {
		return nil
	}
	return rules
}
