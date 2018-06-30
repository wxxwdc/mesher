package config_test

import (
	"github.com/ServiceComb/go-chassis/core/archaius"
	cConfig "github.com/ServiceComb/go-chassis/core/config"
	"github.com/ServiceComb/go-chassis/core/lager"
	"github.com/ServiceComb/go-chassis/util/fileutil"
	"github.com/go-chassis/mesher/cmd"
	"github.com/go-chassis/mesher/config"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
	"testing"
)

func TestGetConfigFilePath(t *testing.T) {
	cmd.Init()
	f, _ := config.GetConfigFilePath()
	assert.Contains(t, f, "mesher.yaml")
}

var file = []byte(`
pprof:
  enable: false
  listen: 0.0.0.0:6060
plugin:
  destinationResolver: host #用户可自定义如何将host转为换destination name，默认为host直接就是sn，
  souceResolver: servicecenter #从source ip反向查询service name
  #这里表示查询api 得到ip所对应的serviceName.namespace.dnsSuffix，对于servicecenter 由于可能是多个微服务在一个host上这种场景无法支持。也不合理，我们干脆绑定一个网络平面的docker
  `)

func TestSetConfig(t *testing.T) {
	c := &config.MesherConfig{}
	if err := yaml.Unmarshal([]byte(file), c); err != nil {
		t.Error(err)
	}
	assert.Equal(t, "host", c.Plugin.DestinationResolver)
}

func TestInit(t *testing.T) {
	s, _ := fileutil.GetWorkDir()
	os.Setenv(fileutil.ChassisHome, s)
	chassisConf := filepath.Join(os.Getenv(fileutil.ChassisHome), "conf")
	os.MkdirAll(chassisConf, 0600)
	f, err := os.Create(filepath.Join(chassisConf, "chassis.yaml"))
	t.Log(f.Name())
	assert.NoError(t, err)
	f, err = os.Create(filepath.Join(chassisConf, "microservice.yaml"))
	t.Log(f.Name())
	assert.NoError(t, err)
	err = cConfig.Init()
	f, err = os.Create(filepath.Join(chassisConf, "mesher.yaml"))
	t.Log(f.Name())
	f.Write(file)
	f.Close()
	lager.Initialize("", "INFO", "", "size", true, 1, 10, 7)
	archaius.Init()

	err = config.Init()
	assert.NoError(t, err)
	t.Log(config.GetConfig())
	assert.Equal(t, "host", config.GetConfig().Plugin.DestinationResolver)
	assert.Equal(t, false, config.GetConfig().PProf.Enable)
	assert.Equal(t, "0.0.0.0:6060", config.GetConfig().PProf.Listen)
}
