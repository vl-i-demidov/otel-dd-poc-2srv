package config

// Config embeds all the configuration related structure.
// https://pkg.go.dev/github.com/mitchellh/mapstructure#DecoderConfig.Squash
type Config struct {
	HttpPort        int       `mapstructure:"HTTP_PORT"`
	ForwardEndpoint string    `mapstructure:"FORWARD_ENDPOINT"`
	Tracing         Tracing   `mapstructure:",squash"`
	Profiling       Profiling `mapstructure:",squash"`
}

const (
	TracingNone = "NONE"
	TracingOTEL = "OTEL"
	TracingDD   = "DD"
)

type Tracing struct {
	Protocol         string `mapstructure:"TRACING_PROTOCOL"` // OTEL | DATADOG | NONE
	ReceiverEndpoint string `mapstructure:"TRACING_RECEIVER_ENDPOINT"`
	Environment      string `mapstructure:"TRACING_ENVIRONMENT"`
	Service          string `mapstructure:"TRACING_SERVICE"`
	AppVersion       string `mapstructure:"TRACING_APP_VERSION"`
}

type Profiling struct {
	Enabled bool `mapstructure:"PROFILING_ENABLED"`
}

type Tracing2 struct {
	ReceiverEndpoint string `mapstructure:"TRACING_RECEIVER_ENDPOINT"`

	// TODO: this should be in a common config
	Environment string `mapstructure:"ENVIRONMENT"`
	Service     string `mapstructure:"SERVICE"`
	AppVersion  string `mapstructure:"APP_VERSION"`
}
