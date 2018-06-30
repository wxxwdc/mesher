package register

import (
	"github.com/go-chassis/mesher/common"
	chassisCommon "github.com/ServiceComb/go-chassis/core/common"
	"github.com/ServiceComb/go-chassis/core/config"
	chassisModel "github.com/ServiceComb/go-chassis/core/config/model"
	"github.com/ServiceComb/go-chassis/core/lager"
	"github.com/ServiceComb/go-chassis/core/registry"
	"github.com/ServiceComb/go-chassis/util/iputil"
	"strings"
)

// AdaptEndpoints moves http endpoint to rest endpoint
func AdaptEndpoints() {
	// To be called by services based on CSE SDK,
	// mesher has to register endpoint with rest://ip:port
	oldProtoMap := config.GlobalDefinition.Cse.Protocols
	if _, ok := oldProtoMap[common.HttpProtocol]; !ok {
		return
	}
	if _, ok := oldProtoMap[chassisCommon.ProtocolRest]; ok {
		return
	}

	newProtoMap := make(map[string]chassisModel.Protocol)
	for n, proto := range oldProtoMap {
		if n == common.HttpProtocol {
			continue
		}
		newProtoMap[n] = proto
	}
	newProtoMap[chassisCommon.ProtocolRest] = oldProtoMap[common.HttpProtocol]
	registry.InstanceEndpoints = registry.MakeEndpointMap(newProtoMap)
	for protocol, address := range registry.InstanceEndpoints {
		if address == "" {
			port := strings.Split(newProtoMap[protocol].Listen, ":")
			if len(port) == 2 { //check if port is not specified along with ip address, eventually in case port is not specified, server start will fail in subsequent processing.
				registry.InstanceEndpoints[protocol] = iputil.GetLocalIP() + ":" + port[1]
			}
		}
	}

	lager.Logger.Debug("Adapt endpoints success")
}
