package metrics

import (
	"net/http"
	"os"
	"sync"
	"time"

	mesherconfig "github.com/go-chassis/mesher/config"

	"github.com/ServiceComb/go-chassis/core/config"
	"github.com/ServiceComb/go-chassis/core/invocation"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	TotalRequest          = "requests_total"
	TotalSuccess          = "successes_total"
	TotalFailures         = "failures_total"
	RequestLatencySeconds = "request_latency_seconds"
	Error4XX              = "status_4xx"
	Error5XX              = "status_5xx"
	ServiceName           = "servicename"
	AppID                 = "appid"
	Version               = "version"
)

var (
	LabelNames = []string{ServiceName, AppID, Version}
	mutex      = sync.Mutex{}
)

var onceEnable sync.Once

func Init() {
	mesherLabelValues := map[string]string{ServiceName: config.SelfServiceName, AppID: config.GlobalDefinition.AppID, Version: config.SelfVersion}
	mesherStartTime := time.Now().Unix()
	DefaultPrometheusExporter.Gauge("start_time_seconds", float64(mesherStartTime), LabelNames, mesherLabelValues)
	mesherConfig := mesherconfig.GetConfig()
	promConfig := getPrometheusSinker(getSystemPrometheusRegistry())
	if mesherConfig.Admin.GoRuntimeMetrics == true {
		onceEnable.Do(func() {
			promConfig.PromRegistry.MustRegister(prometheus.NewProcessCollector(os.Getpid(), ""))
			promConfig.PromRegistry.MustRegister(prometheus.NewGoCollector())
		})
	}
}
func RecordResponse(inv *invocation.Invocation, statusCode int) {
	mutex.Lock()
	defer mutex.Unlock()
	serviceLabelValues := map[string]string{ServiceName: inv.MicroServiceName, AppID: inv.AppID, Version: inv.Version}
	if statusCode >= http.StatusBadRequest && statusCode <= http.StatusUnavailableForLegalReasons {
		DefaultPrometheusExporter.Count(Error4XX, LabelNames, serviceLabelValues)
		DefaultPrometheusExporter.Count(TotalFailures, LabelNames, serviceLabelValues)
	} else if statusCode >= http.StatusInternalServerError && statusCode <= http.StatusNetworkAuthenticationRequired {
		DefaultPrometheusExporter.Count(Error5XX, LabelNames, serviceLabelValues)
		DefaultPrometheusExporter.Count(TotalFailures, LabelNames, serviceLabelValues)
	} else if statusCode >= http.StatusOK && statusCode <= http.StatusIMUsed {
		DefaultPrometheusExporter.Count(TotalSuccess, LabelNames, serviceLabelValues)
	}

	DefaultPrometheusExporter.Count(TotalRequest, LabelNames, serviceLabelValues)
}
