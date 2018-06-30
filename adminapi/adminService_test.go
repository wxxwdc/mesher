package adminapi

import (
	"errors"
	"github.com/ServiceComb/go-chassis/core/config"
	"github.com/ServiceComb/go-chassis/core/config/model"
	"github.com/ServiceComb/go-chassis/core/lager"
	"github.com/ServiceComb/go-chassis/core/registry"
	"github.com/ServiceComb/go-chassis/core/registry/mock"
	"github.com/ServiceComb/go-chassis/core/router"
	_ "github.com/ServiceComb/go-chassis/core/router/cse"
	"github.com/emicklei/go-restful"
	routerMock "github.com/go-chassis/mesher/adminapi/route/mock"
	ver "github.com/go-chassis/mesher/adminapi/version"
	mesherconfig "github.com/go-chassis/mesher/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"net/http"
	"net/http/httptest"
	"testing"
	//"github.com/ServiceComb/go-chassis/server/restful"
)

var globalConfig = `
---
APPLICATION_ID: sockshop
cse:
  loadbalance:
    strategyName: RoundRobin
  service:
    registry:
      address: http://10.162.197.14:30100
      scope: full
      watch: true
  protocols:
    http:
      listenAddress: 127.0.0.1:30101
  handler:
    chain:
      consumer:
        income:  ratelimiter-provider,local-selection
`
var mesherConf = `
routeRule:
  ShoppingCart:
    - precedence: 2
      route:
      - tags:
          version: 1.2
          app: HelloWorld
        weight: 80
      - tags:
          version: 1.3
          app: HelloWorld
        weight: 20
      match:
        refer: vmall-with-special-header
        source: vmall
        sourceTags:
            version: v2
        httpHeaders:
            cookie:
              regex: "^(.*?;)?(user=jason)(;.*)?$"
            X-Age:
              exact: "18"
    - precedence: 1
      route:
      - tags:
          version: 1.0
        weight: 100
`

func TestAdminService_GetVersion(t *testing.T) {
	t.Log("testing /v1/mesher/version admin api of mesher")
	assert := assert.New(t)

	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/v1/mesher/version", nil)
	require.Nil(t, err)
	rr := httptest.NewRecorder()

	restfulRequest := restful.NewRequest(req)
	restfulResponse := restful.NewResponse(rr)
	GetVersion(restfulRequest, restfulResponse)

	t.Log("Response : ", rr.Body.String())
	assert.Equal(rr.Code, http.StatusOK)
	assert.NotEmpty(rr.Body.String())

}

func TestAdminService_MesherHealthAPI(t *testing.T) {
	t.Log("testing /v1/mesher/health admin api of mesher when instance of mesher is present in SC")
	assert := assert.New(t)
	config.GlobalDefinition = new(model.GlobalCfg)
	yaml.Unmarshal([]byte(globalConfig), config.GlobalDefinition)
	config.SelfServiceName = "mesher"
	config.MicroserviceDefinition = &model.MicroserviceCfg{}
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/v1/mesher/health", nil)
	require.Nil(t, err)
	rr := httptest.NewRecorder()

	mockinstances := []*registry.MicroServiceInstance{
		&registry.MicroServiceInstance{
			InstanceID: "testInstanceId",
			ServiceID:  "testMicroserviceId",
		},
	}
	testRegistryObj := new(mock.RegistratorMock)
	testDiscoveryObj := new(mock.DiscoveryMock)
	registry.DefaultRegistrator = testRegistryObj
	registry.DefaultServiceDiscoveryService = testDiscoveryObj
	testDiscoveryObj.On("GetMicroServiceID", "sockshop", "mesher", ver.DefaultVersion, "").Return("testMicroserviceId", nil)
	testDiscoveryObj.On("GetMicroServiceInstances", "testMicroserviceId", "testMicroserviceId").Return(mockinstances, nil)
	testRegistryObj.On("Heartbeat", "testMicroserviceId", "testInstanceId").Return(true, nil)

	restfulRequest := restful.NewRequest(req)
	restfulResponse := restful.NewResponse(rr)
	MesherHealth(restfulRequest, restfulResponse)

	t.Log("Response : ", rr.Body.String())
	t.Log("Response Status Code : ", rr.Code)
	assert.Equal(rr.Code, http.StatusOK)
}

func TestAdminService_MesherHealth2(t *testing.T) {
	t.Log("testing /v1/mesher/health admin api of mesher when no instance of mesher is not present in SC")
	assert := assert.New(t)
	lager.Initialize("", "INFO", "", "size", true, 1, 10, 7)
	config.GlobalDefinition = new(model.GlobalCfg)
	yaml.Unmarshal([]byte(globalConfig), config.GlobalDefinition)

	config.SelfServiceName = "mesher"
	config.MicroserviceDefinition = &model.MicroserviceCfg{}

	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/v1/mesher/health", nil)
	require.Nil(t, err)
	rr := httptest.NewRecorder()

	testDiscoveryObj := new(mock.DiscoveryMock)
	registry.DefaultServiceDiscoveryService = testDiscoveryObj
	testDiscoveryObj.On("GetMicroServiceID", "sockshop", "mesher", ver.DefaultVersion, "").Return("", errors.New("MOCK_ERROR"))

	restfulRequest := restful.NewRequest(req)
	restfulResponse := restful.NewResponse(rr)
	MesherHealth(restfulRequest, restfulResponse)

	t.Log("Response : ", rr.Body.String())
	t.Log("Response Status Code : ", rr.Code)
	assert.Equal(rr.Code, http.StatusInternalServerError)

}

func TestAdminService_RouteRule(t *testing.T) {
	t.Log("testing /v1/mesher/routeRule admin api of mesher ")
	ast := assert.New(t)
	lager.Initialize("", "INFO", "", "size", true, 1, 10, 7)
	mesherConfig := new(model.RouterConfig)
	yaml.Unmarshal([]byte(mesherConf), mesherConfig)
	testRouterMock := &routerMock.RouterMock{}
	router.DefaultRouter = testRouterMock

	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/v1/mesher/routeRule", nil)
	require.Nil(t, err)
	rr := httptest.NewRecorder()

	restfulRequest := restful.NewRequest(req)
	restfulResponse := restful.NewResponse(rr)
	RouteRule(restfulRequest, restfulResponse)

	t.Log("Response : ", rr.Body.String())
	t.Log("Response Status Code : ", rr.Code)
	ast.Equal(rr.Code, http.StatusOK)

	t.Log("testing /v1/mesher/routeRule/{serviceName} admin api of mesher when given service route rule is present")
	req, err = http.NewRequest(http.MethodGet, "http://localhost:8080/v1/mesher/routeRule/ShoppingCart", nil)
	require.Nil(t, err)
	query := req.URL.Query()
	query.Add(":serviceName", "ShoppingCart")
	req.URL.RawQuery = query.Encode()
	rr = httptest.NewRecorder()
	testRouterMock.On("FetchRouteRuleByServiceName", "ShoppingCart").Return(mesherConfig.Destinations["ShoppingCart"])

	restfulRequest = restful.NewRequest(req)
	restfulResponse = restful.NewResponse(rr)
	RouteRuleByService(restfulRequest, restfulResponse)

	t.Log("Response : ", rr.Body.String())
	t.Log("Response Status Code : ", rr.Code)
	ast.Equal(rr.Code, http.StatusOK)

	t.Log("testing /v1/mesher/routeRule/{serviceName} admin api of mesher when given service route rule is not present")

	req, err = http.NewRequest(http.MethodGet, "http://localhost:8080/v1/mesher/routeRule/Invalid", nil)
	require.Nil(t, err)
	query = req.URL.Query()
	query.Add(":serviceName", "Invalid")
	req.URL.RawQuery = query.Encode()
	rr = httptest.NewRecorder()
	testRouterMock.On("FetchRouteRuleByServiceName", "Invalid").Return(nil)

	restfulRequest = restful.NewRequest(req)
	restfulResponse = restful.NewResponse(rr)
	RouteRuleByService(restfulRequest, restfulResponse)

	t.Log("Response : ", rr.Body.String())
	t.Log("Response Status Code : ", rr.Code)
	ast.Equal(rr.Code, http.StatusNotFound)
}

func TestInit(t *testing.T) {
	t.Log("testing mesher admin protocol when protocol URI is valid")
	assert := assert.New(t)
	lager.Initialize("", "INFO", "", "size", true, 1, 10, 7)
	mesherConfig := new(mesherconfig.MesherConfig)
	yaml.Unmarshal([]byte(mesherConf), mesherConfig)
	mesherconfig.SetConfig(mesherConfig)
	err := Init()
	assert.Nil(err)
}

func TestInit2(t *testing.T) {
	t.Log("testing mesher admin protocol when protocol URI is not valid")
	assert := assert.New(t)
	mesherConfig := new(mesherconfig.MesherConfig)
	yaml.Unmarshal([]byte(mesherConf), mesherConfig)
	mesherConfig.Admin.ServerUri = "INVALID"
	mesherconfig.SetConfig(mesherConfig)
	err := Init()
	assert.NotNil(err)
}
