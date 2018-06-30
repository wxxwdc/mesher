package metrics

import (
	"net/http"
	"strings"
	"testing"

	"github.com/ServiceComb/go-chassis/core/config"
	"github.com/ServiceComb/go-chassis/core/invocation"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"

	"github.com/ServiceComb/go-chassis/core/config/model"
)

var (
	labelNames  = []string{"APPID", "VERSION"}
	labelValues = map[string]string{"APPID": "sockshop", "VERSION": "0.1"}
)

func TestPrometheusConfig_CounterFromNameAndLabelValues(t *testing.T) {
	assert := assert.New(t)
	var totalMetricCreated int
	DefaultPrometheusExporter.Count("total_request", labelNames, labelValues)
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	assert.Nil(err, "error should be nil while collecting metrics from prometheus")
	for _, metricFamily := range metricFamilies {
		if metricName := metricFamily.GetName(); strings.Contains(metricName, "total_request") {
			assert.Equal(metricFamily.GetType(), dto.MetricType_COUNTER)
			totalMetricCreated++
		}
	}
	assert.Equal(totalMetricCreated, 1)
}

func TestPrometheusConfig_GaugeFromNameAndLabelValues(t *testing.T) {
	assert := assert.New(t)
	var totalMetricCreated int
	var gaugeValue *float64
	DefaultPrometheusExporter.Gauge("memory_used", 12, labelNames, labelValues)
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	assert.Nil(err, "error should be nil while collecting metrics from prometheus")
	for _, metricFamily := range metricFamilies {
		if metricName := metricFamily.GetName(); strings.Contains(metricName, "memory_used") {
			assert.Equal(metricFamily.GetType(), dto.MetricType_GAUGE)
			totalMetricCreated++
			gaugeValue = metricFamily.Metric[0].Gauge.Value
		}
	}
	assert.Equal(totalMetricCreated, 1)
	assert.Equal(*gaugeValue, float64(12))
}

func TestPrometheusConfig_SummaryFromNameAndLabelValues(t *testing.T) {
	assert := assert.New(t)
	var totalMetricCreated int
	var sampleCount *uint64
	DefaultPrometheusExporter.Summary("request_latency", 12, labelNames, labelValues)
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	assert.Nil(err, "error should be nil while collecting metrics from prometheus")
	for _, metricFamily := range metricFamilies {
		if metricName := metricFamily.GetName(); strings.Contains(metricName, "request_latency") {
			assert.Equal(metricFamily.GetType(), dto.MetricType_SUMMARY)
			totalMetricCreated++
			sampleCount = metricFamily.Metric[0].Summary.SampleCount
		}
	}
	assert.Equal(totalMetricCreated, 1)
	assert.Equal(*sampleCount, uint64(1))
}

func TestPrepare(t *testing.T) {
	assert := assert.New(t)
	config.GlobalDefinition = new(model.GlobalCfg)
	config.GlobalDefinition.AppID = "sockshop"
	config.SelfVersion = "0.1"
	var inv = &invocation.Invocation{
		MicroServiceName: "service",
	}
	var errorcount4xx float64
	var errorcount5xx float64
	RecordResponse(inv, http.StatusOK)
	RecordResponse(inv, http.StatusNotFound)
	RecordResponse(inv, http.StatusInternalServerError)
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	assert.Nil(err, "error should be nil while collecting metrics from prometheus")
	for _, metricFamily := range metricFamilies {
		if name := metricFamily.GetName(); strings.Contains(name, Error5XX) {
			errorcount4xx += *metricFamily.Metric[0].Counter.Value
		}
	}
	for _, metricFamily := range metricFamilies {
		if name := metricFamily.GetName(); strings.Contains(name, Error5XX) {
			errorcount5xx += *metricFamily.Metric[0].Counter.Value
		}
	}
	assert.Equal(errorcount4xx, float64(1))
	assert.Equal(errorcount5xx, float64(1))

}
