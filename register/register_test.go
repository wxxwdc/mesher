package register

import (
	"github.com/go-chassis/mesher/common"
	chassisCommon "github.com/ServiceComb/go-chassis/core/common"
	"github.com/ServiceComb/go-chassis/core/config"
	"github.com/ServiceComb/go-chassis/core/config/model"
	"github.com/ServiceComb/go-chassis/core/lager"
	"github.com/ServiceComb/go-chassis/core/registry"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAdaptEndpoints(t *testing.T) {
	lager.Initialize("", "INFO", "", "size", true, 1, 10, 7)
	protoMap := make(map[string]model.Protocol)
	config.GlobalDefinition = &model.GlobalCfg{
		Cse: model.CseStruct{
			Protocols: protoMap,
		},
	}

	AdaptEndpoints()
	assert.Nil(t, registry.InstanceEndpoints)

	protoMap[chassisCommon.ProtocolRest] = model.Protocol{
		Advertise: "1.1.1.1:8080",
	}
	AdaptEndpoints()
	assert.Nil(t, registry.InstanceEndpoints)

	protoMap[common.HttpProtocol] = model.Protocol{
		Advertise: "1.1.1.1:8081",
	}
	delete(protoMap, chassisCommon.ProtocolRest)
	AdaptEndpoints()
	assert.Equal(t, 1, len(registry.InstanceEndpoints))
	_, ok := registry.InstanceEndpoints[common.HttpProtocol]
	assert.False(t, ok)
	endpoint0 := registry.InstanceEndpoints[chassisCommon.ProtocolRest]
	endpoint1 := protoMap[common.HttpProtocol].Advertise
	assert.Equal(t, endpoint0, endpoint1)
}
