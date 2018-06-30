package route

import (
	"github.com/ServiceComb/go-chassis/core/config/model"
	"github.com/ServiceComb/go-chassis/core/router"
)

type Rules struct {
	Destinations map[string][]*model.RouteRule `yaml:"routeRule"`
}

var routeRules *Rules

func GetRouteRules() *Rules {
	if routeRules != nil {
		return routeRules
	}
	routeRules = new(Rules)
	routeRules.Destinations = router.DefaultRouter.FetchRouteRule()
	return routeRules
}

func GetServiceRouteRule(serviceName string) []*model.RouteRule {
	routeRules := GetRouteRules()
	routeRule, ok := routeRules.Destinations[serviceName]
	if !ok {
		return nil
	}
	return routeRule
}
