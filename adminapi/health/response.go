package health

type StatusCode string

const (
	Red   StatusCode = "red"
	Green StatusCode = "green"
)

type Health struct {
	ServiceName                 string     `json:"serviceName,omitempty"`
	Version                     string     `json:"version,omitempty"`
	Status                      StatusCode `json:"status,omitempty"`
	ConnectedConfigCenterClient bool       `json:"connectedConfigCenterClient"`
	ConnectedMonitoring         bool       `json:"connectedMonitoring"`
	Error                       string     `json:"error,omitempty"`
}
