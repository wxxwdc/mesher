package resolver_test

import (
	"github.com/go-chassis/mesher/common"
	"github.com/go-chassis/mesher/config"
	"github.com/go-chassis/mesher/resolver"
	chassisConfig "github.com/ServiceComb/go-chassis/core/config"
	chassisConfigModel "github.com/ServiceComb/go-chassis/core/config/model"
	"github.com/ServiceComb/go-chassis/core/lager"
	"github.com/ServiceComb/go-chassis/core/registry"
	"github.com/ServiceComb/go-sc-client/model"
	"github.com/stretchr/testify/assert"
	"runtime"
	"testing"
)

func TestDefaultSourceResolver_Resolve(t *testing.T) {
	lager.Initialize("", "DEBUG", "", "size", true, 1, 10, 7)
	ms := &model.MicroService{
		ServiceName: "testService",
		AppID:       "testApp",
		Version:     "1.0.0",
		Properties: map[string]string{
			"author": "zhangsan",
			"region": "China",
		},
	}
	var nilMS *model.MicroService = nil
	sourceIp := "1.2.3.4"
	sourceIP2 := "1.1.1.1"
	registry.IPIndexedCache.Set(sourceIp, ms, 0)
	registry.IPIndexedCache.Set(sourceIP2, nilMS, 0)

	sr := resolver.GetSourceResolver()
	sourceInfo := sr.Resolve(sourceIp)
	assert.NotNil(t, sourceInfo)
	sourceInfo2 := sr.Resolve(sourceIP2)
	assert.Nil(t, sourceInfo2)
	assert.Equal(t, ms.ServiceName, sourceInfo.Name)
	assert.Equal(t, ms.AppID, sourceInfo.Tags[common.BuildInTagApp])
	assert.Equal(t, ms.Version, sourceInfo.Tags[common.BuildInTagVersion])
	assert.Equal(t, len(ms.Properties), len(sourceInfo.Tags)-2)
	for k, v := range ms.Properties {
		assert.Equal(t, v, sourceInfo.Tags[k])
	}
	t.Log("resolve local request")
	sourceInfo = sr.Resolve("127.0.0.1")
	assert.Nil(t, sourceInfo)
	t.Log("resolve local request in sidecar")
	chassisConfig.SelfVersion = ms.Version
	chassisConfig.SelfServiceName = ms.ServiceName
	chassisConfig.SelfMetadata = map[string]string{
		"author": "zhangsan",
		"region": "China",
	}
	chassisConfig.GlobalDefinition = &chassisConfigModel.GlobalCfg{}
	chassisConfig.GlobalDefinition.AppID = ms.AppID
	config.Mode = common.ModeSidecar
	sourceInfo = sr.Resolve("127.0.0.1")
	assert.Nil(t, sourceInfo)

	//error case
	sourceIp1 := ""
	sr = resolver.GetSourceResolver()
	runtime.Gosched()
	sourceInfo = sr.Resolve(sourceIp1)
	assert.Nil(t, sourceInfo)

	//error case
	sourceIp2 := "1.2.3.4"
	ms1 := &resolver.DefaultSourceResolver{}
	registry.IPIndexedCache.Set(sourceIp2, ms1, 0)

	sourceInfo = sr.Resolve(sourceIp2)
	assert.Nil(t, sourceInfo)

}
