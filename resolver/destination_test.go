package resolver

import (
	"testing"

	"github.com/ServiceComb/go-chassis/core/archaius"
	cConfig "github.com/ServiceComb/go-chassis/core/config"
	"github.com/ServiceComb/go-chassis/core/lager"
	"github.com/ServiceComb/go-chassis/util/fileutil"
	"github.com/go-chassis/mesher/cmd"
	"github.com/go-chassis/mesher/config"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"os"
	"path/filepath"
)

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
	err = cmd.Init()
	lager.Initialize("", "INFO", "", "size", true, 1, 10, 7)
	archaius.Init()
	config.Init()
	err = Init()
	assert.NoError(t, err)
}

func TestResolve(t *testing.T) {
	d := &DefaultDestinationResolver{}
	header := fasthttp.RequestHeader{}
	header.Add("cookie", "user=jason")
	header.Add("X-Age", "18")
	mystring := "Server"
	var destinationString *string = &mystring
	err := d.Resolve("abc", map[string]string{}, "127.0.1.1", destinationString)
	assert.Error(t, err)
	err = d.Resolve("abc", map[string]string{}, "", destinationString)
	assert.Error(t, err)
	err = d.Resolve("abc", map[string]string{}, "http://127.0.0.1:80/test", destinationString)
	assert.NoError(t, err)
}

func TestGetDestinationResolver(t *testing.T) {
	GetDestinationResolver()
}
