package config

type MesherConfig struct {
	PProf      *PProf  `yaml:"pprof"`
	Plugin     *Plugin `yaml:"plugin"`
	Admin      Admin   `yaml:"admin"`
	ProxyedPro string  `yaml:"proxyedProtocol"`
}
type PProf struct {
	Enable bool   `yaml:"enable"`
	Listen string `yaml:"listen"`
}
type Policy struct {
	Destination   string            `yaml:"destination"`
	Tags          map[string]string `yaml:"tags"`
	LoadBalancing map[string]string `yaml:"loadBalancing"`
}
type Plugin struct {
	DestinationResolver string `yaml:"destinationResolver"`
	SourceResolver      string `yaml:"sourceResolver"`
}

type Admin struct {
	Enable           *bool  `yaml:"enable"`
	ServerUri        string `yaml:"serverUri"`
	GoRuntimeMetrics bool   `yaml:"goRuntimeMetrics"`
}
