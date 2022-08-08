package config

// Config embeds all the configuration related structure.
// https://pkg.go.dev/github.com/mitchellh/mapstructure#DecoderConfig.Squash
type Config struct {
	HttpPort        int       `mapstructure:"HTTP_PORT"`
	ForwardEndpoint string    `mapstructure:"FORWARD_ENDPOINT"`
	Tracing         Tracing   `mapstructure:",squash"`
	Profiling       Profiling `mapstructure:",squash"`
}

type Tracing struct {
	ReceiverEndpoint string  `mapstructure:"TRACING_RECEIVER_ENDPOINT"`
	SamplingRatio    float64 `mapstructure:"TRACING_SAMPLING_RATIO"`

	// TODO: this should be in a common config
	Environment string `mapstructure:"ENVIRONMENT"`
	Service     string `mapstructure:"SERVICE"`
	AppVersion  string `mapstructure:"APP_VERSION"`
}

type Profiling struct {
	Enabled bool `mapstructure:"PROFILING_ENABLED"`
}
