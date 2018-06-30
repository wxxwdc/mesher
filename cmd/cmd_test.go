package cmd_test

import (
	"github.com/ServiceComb/go-chassis/core/lager"
	"github.com/go-chassis/mesher/cmd"
	"github.com/go-chassis/mesher/common"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestParseConfigFromCmd(t *testing.T) {
	config := "/mesher.yaml"

	t.Log("========cmd --config=", config)
	os.Args = []string{"test", "--config", config}
	err := cmd.Init()
	configFromCmd := cmd.Configs
	assert.Equal(t, config, configFromCmd.ConfigFile)
	assert.Nil(t, err)

}

func TestConfigFromCmd_GeneratePortsMap(t *testing.T) {
	lager.Initialize("", "DEBUG", "", "size", true, 1, 10, 7)

	c := &cmd.ConfigFromCmd{
		LocalServicePorts: "rest:80,grpc:8000",
	}
	c.GeneratePortsMap()
	t.Log(c.PortsMap)
	assert.Equal(t, "127.0.0.1:80", c.PortsMap["rest"])
}
func TestConfigFromCmd_GeneratePortsMap2(t *testing.T) {
	lager.Initialize("", "DEBUG", "", "size", true, 1, 10, 7)

	c := &cmd.ConfigFromCmd{
		LocalServicePorts: "rest: 80,grpc",
	}
	err := c.GeneratePortsMap()
	t.Log(c.PortsMap)
	assert.Error(t, err)
}
func TestConfigFromCmd_GeneratePortsMap3(t *testing.T) {
	lager.Initialize("", "DEBUG", "", "size", true, 1, 10, 7)
	os.Setenv(common.EnvServicePorts, "rest:80,grpc:90")
	cmd.Init()
	_ = cmd.Configs.GeneratePortsMap()
	t.Log(cmd.Configs.PortsMap)
	assert.Equal(t, "127.0.0.1:80", cmd.Configs.PortsMap["rest"])
}
func TestConfigFromCmd_GeneratePortsMap4(t *testing.T) {
	lager.Initialize("", "DEBUG", "", "size", true, 1, 10, 7)
	os.Setenv(common.EnvSpecificAddr, "127.0.0.1:80")
	cmd.Init()
	_ = cmd.Configs.GeneratePortsMap()
	t.Log(cmd.Configs.PortsMap)
	assert.Equal(t, "127.0.0.1:80", cmd.Configs.PortsMap["rest"])
}
