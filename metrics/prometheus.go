package metrics

import (
	"github.com/ServiceComb/go-chassis/core/lager"
	"github.com/ServiceComb/go-chassis/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"runtime"
	"sync"
)

type PrometheusExporter struct {
	gaugesMutex   sync.RWMutex
	countersMutex sync.RWMutex
	summaryMutex  sync.RWMutex
	registry      *prometheus.Registry
	gauges        map[string]*prometheus.GaugeVec
	counters      map[string]*prometheus.CounterVec
	summary       map[string]*prometheus.SummaryVec
}

// PrometheusMesherSinker is the struct for prometheus configuration parameters
type PrometheusMesherSinker struct {
	PromRegistry prometheus.Registerer //Prometheus registry
}

var (
	DefaultPrometheusExporter = GetPrometheusExporter()
	prometheusRegistry        = prometheus.NewRegistry()
	onceInit                  sync.Once
	DefaultPrometheusSinker   *PrometheusMesherSinker
)

//GetSystemPrometheusRegistry return prometheus registry which mesher use
func getSystemPrometheusRegistry() *prometheus.Registry {
	return prometheusRegistry
}

func newPrometheusProvider(promRegistry prometheus.Registerer) *PrometheusMesherSinker {
	return &PrometheusMesherSinker{
		PromRegistry: promRegistry,
	}
}

func getPrometheusSinker(pr *prometheus.Registry) *PrometheusMesherSinker {
	onceInit.Do(func() {
		DefaultPrometheusSinker = newPrometheusProvider(pr)
	})
	return DefaultPrometheusSinker
}

func GetPrometheusExporter() *PrometheusExporter {
	//use go chassis registry
	var promRegistry = metrics.GetSystemPrometheusRegistry()
	prometheus.DefaultGatherer = promRegistry
	prometheus.DefaultRegisterer = promRegistry
	return &PrometheusExporter{
		registry:      promRegistry,
		gauges:        make(map[string]*prometheus.GaugeVec),
		counters:      make(map[string]*prometheus.CounterVec),
		summary:       make(map[string]*prometheus.SummaryVec),
		summaryMutex:  sync.RWMutex{},
		gaugesMutex:   sync.RWMutex{},
		countersMutex: sync.RWMutex{},
	}
}

func (s *PrometheusExporter) Count(name string, labelNames []string, labels prometheus.Labels) {
	s.countersMutex.RLock()
	cv, ok := s.counters[name]
	s.countersMutex.RUnlock()
	if !ok {
		cv = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: name,
			Help: name,
		}, labelNames)
		s.registry.MustRegister(cv)
		s.countersMutex.Lock()
		s.counters[name] = cv
		defer s.countersMutex.Unlock()
	}
	cv.With(labels).Add(1)

}

func (s *PrometheusExporter) Gauge(name string, val float64, labelNames []string, labels prometheus.Labels) {
	defer recoverPanic(name)
	s.gaugesMutex.RLock()
	g, ok := s.gauges[name]
	s.gaugesMutex.RUnlock()
	if !ok {
		g = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: name,
			Help: name,
		}, labelNames)
		s.registry.MustRegister(g)
		s.gaugesMutex.Lock()
		s.gauges[name] = g
		defer s.gaugesMutex.Unlock()
	}
	g.With(labels).Set(val)
}

func (s *PrometheusExporter) Summary(name string, val float64, labelNames []string, labels prometheus.Labels) {
	defer recoverPanic(name)
	s.summaryMutex.RLock()
	sm, ok := s.summary[name]
	s.summaryMutex.RUnlock()
	if !ok {
		sm = prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Name: name,
			Help: name,
		}, labelNames)
		s.registry.MustRegister(sm)
		s.summaryMutex.Lock()
		s.summary[name] = sm
		defer s.summaryMutex.Unlock()
	}
	sm.With(labels).Observe(val)
}

func recoverPanic(metricName string) {
	if r := recover(); r != nil {
		pc := make([]uintptr, 10)
		runtime.Callers(1, pc)
		lager.Logger.Warnf("panics while registering metric [%s] to prometheus %s", metricName, r)
	}
}
