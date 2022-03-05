package extension

import (
	"os"

	suffiksv1 "github.com/suffiks/suffiks/api/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type Config interface {
	getListenAddress() string
	getTracing() suffiksv1.TracingConfig
}

type ConfigSpec struct {
	// ListenAddress is the address to listen on for the extension.
	// Defaults to :4269
	ListenAddress string `json:"listenAddress"`

	// Tracing is used to configure tracing exporter.
	// +optional
	Tracing suffiksv1.TracingConfig `json:"tracing"`
}

func (c *ConfigSpec) getListenAddress() string {
	if c.ListenAddress == "" {
		return ":4269"
	}

	return c.ListenAddress
}

func (c *ConfigSpec) getTracing() suffiksv1.TracingConfig {
	return c.Tracing
}

func (c ConfigSpec) extensionConfig() {}

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
