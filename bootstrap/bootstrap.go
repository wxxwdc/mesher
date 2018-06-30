package bootstrap

import (
	"log"
	"strings"

	"github.com/go-chassis/mesher/adminapi"
	"github.com/go-chassis/mesher/adminapi/version"
	"github.com/go-chassis/mesher/cmd"
	"github.com/go-chassis/mesher/common"
	"github.com/go-chassis/mesher/config"
	"github.com/go-chassis/mesher/handler"
	"github.com/go-chassis/mesher/register"
	"github.com/go-chassis/mesher/resolver"

	"github.com/ServiceComb/go-chassis"
	chassisHandler "github.com/ServiceComb/go-chassis/core/handler"
	"github.com/ServiceComb/go-chassis/core/lager"
	"github.com/ServiceComb/go-chassis/core/metadata"
)

// Start initialize configs and components
func Start() error {
	if err := config.InitProtocols(); err != nil {
		return err
	}
	if err := config.Init(); err != nil {
		return err
	}
	if err := resolver.Init(); err != nil {
		return err
	}
	if err := DecideMode(); err != nil {
		return err
	}
	if err := adminapi.Init(); err != nil {
		log.Println("Error occured in starting admin server", err)
	}
	register.AdaptEndpoints()
	if cmd.Configs.LocalServicePorts == "" {
		lager.Logger.Warnf("local service ports is missing, service can not be called by mesher")
	} else {
		lager.Logger.Infof("local service ports is [%v]", cmd.Configs.PortsMap)
	}

	return nil

}

func DecideMode() error {
	config.Mode = cmd.Configs.Mode
	lager.Logger.Info("Running as "+config.Mode, nil)
	return nil
}

func RegisterFramework() {
	if framework := metadata.NewFramework(); cmd.Configs.Mode == common.ModeSidecar {
		version := GetVersion()
		framework.SetName("Mesher")
		framework.SetVersion(version)
		framework.SetRegister("SIDECAR")
	} else if cmd.Configs.Mode == common.ModePerHost {
		framework.SetName("Mesher")
	}
}

func GetVersion() string {
	versionId := version.Ver().Version
	if len(versionId) == 0 {
		return version.DefaultVersion
	}
	return versionId
}

//SetHandlers leverage go-chassis API to set default handlers if there is no define in chassis.yaml
func SetHandlers() {
	consumerChain := strings.Join([]string{
		chassisHandler.Router,
		chassisHandler.RatelimiterConsumer,
		chassisHandler.BizkeeperConsumer,
		chassisHandler.Loadbalance,
		chassisHandler.Transport,
	}, ",")
	providerChain := strings.Join([]string{
		chassisHandler.RatelimiterProvider,
		chassisHandler.BizkeeperProvider,
		handler.LocalSelection,
		handler.XForward,
		chassisHandler.Transport,
	}, ",")
	consumerChainMap := map[string]string{
		common.ChainConsumerOutgoing: consumerChain,
	}
	providerChainMap := map[string]string{
		common.ChainProviderIncoming: providerChain,
	}
	chassis.SetDefaultConsumerChains(consumerChainMap)
	chassis.SetDefaultProviderChains(providerChainMap)
}
