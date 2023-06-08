package extension

import (
	"os"

	"k8s.io/apimachinery/pkg/util/yaml"
)

type TracingConfig struct {
	// OTLP GRPC tracing endpoint. If empty, tracing is disabled.
	OTLPEndpoint string `json:"otlpEndpoint,omitempty"`

	// Attributes to be added to all spans.
	Attributes map[string]string `json:"attributes,omitempty"`
}

func (t TracingConfig) Enabled() bool {
	return t.OTLPEndpoint != ""
}

type Config interface {
	getListenAddress() string
	getTracing() TracingConfig
}

type ConfigSpec struct {
	// ListenAddress is the address to listen on for the extension.
	// Defaults to :4269
	ListenAddress string `json:"listenAddress"`

	// Tracing is used to configure tracing exporter.
	// +optional
	Tracing TracingConfig `json:"tracing"`
}

func (c ConfigSpec) getListenAddress() string {
	if c.ListenAddress == "" {
		return ":4269"
	}

	return c.ListenAddress
}

func (c ConfigSpec) getTracing() TracingConfig {
	return c.Tracing
}

func ReadConfig(filePath string, v Config) error {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(b, v); err != nil {
		return err
	}

	return nil
}
