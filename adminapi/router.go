package adminapi

import (
	"crypto/tls"
	chassisCom "github.com/ServiceComb/go-chassis/core/common"
	"github.com/ServiceComb/go-chassis/core/lager"
	chassisTLS "github.com/ServiceComb/go-chassis/core/tls"
	"github.com/emicklei/go-restful"
	"github.com/go-chassis/mesher/common"
	"github.com/go-chassis/mesher/config"
	"github.com/go-chassis/mesher/metrics"
	"net"
	"net/http"
	"strings"
	"time"
)

func Init() (err error) {
	var isAdminEnable *bool = config.GetConfig().Admin.Enable

	if isAdminEnable != nil && *isAdminEnable == false {
		lager.Logger.Infof("admin api are not enable")
		return nil
	}

	errCh := make(chan error)
	metrics.Init()

	adminServerUri := config.GetConfig().Admin.ServerUri

	if adminServerUri == "" {
		adminServerUri = "0.0.0.0:30102"
	}
	ln, err := net.Listen("tcp", adminServerUri)
	if err != nil {
		return
	}
	tlsConfig, err := getTlsConfig()
	if err != nil {
		return
	}
	if tlsConfig != nil {
		lager.Logger.Infof("admin server is using ssl")
		ln = tls.NewListener(ln, tlsConfig)
	} else {
		lager.Logger.Infof("admin server is not using ssl")
	}

	go func() {
		lager.Logger.Infof("admin server listening on %s", ln.Addr().String())
		restfulWebService := GetWebService()
		restful.Add(&restfulWebService)
		if err := http.Serve(ln, nil); err != nil {
			errCh <- err
			return
		}
	}()

	select {
	case err = <-errCh:
		lager.Logger.Warnf("got Admin Server Error, err: %v", err)
	case <-time.After(time.Second):
		lager.Logger.Infof("admin server start success")
		err = nil
	}
	return
}

func getTlsConfig() (*tls.Config, error) {
	var tlsConfig *tls.Config
	sslTag := genTag(common.ComponentName, chassisCom.Provider)
	tmpTlsConfig, sslConfig, err := chassisTLS.GetTLSConfigByService(common.ComponentName, "", chassisCom.Provider)
	if err != nil {
		if !chassisTLS.IsSSLConfigNotExist(err) {
			return nil, err
		}
	} else {
		lager.Logger.Warnf("%s TLS mode, verify peer: %t, cipher plugin: %s.",
			sslTag, sslConfig.VerifyPeer, sslConfig.CipherPlugin)
		tlsConfig = tmpTlsConfig
	}
	return tlsConfig, nil
}

func genTag(s ...string) string {
	return strings.Join(s, ".")
}
