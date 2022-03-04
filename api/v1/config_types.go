package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cfg "sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
)

type TracingConfig struct {
	// OTLP GRPC tracing endpoint. If empty, tracing is disabled.
	// +optional
	OTLPEndpoint string `json:"otlpEndpoint,omitempty"`

	// Attributes to be added to all spans.
	// +optional
	Attributes map[string]string `json:"attributes,omitempty"`
}

func (t TracingConfig) Enabled() bool {
	return t.OTLPEndpoint != ""
}

// +kubebuilder:object:root=true

// ProjectConfig is the Schema for the projectconfigs API
type ProjectConfig struct {
	metav1.TypeMeta `json:",inline"`

	// ControllerManagerConfigurationSpec returns the configuration for controllers
	cfg.ControllerManagerConfigurationSpec `json:",inline"`

	// Tracing is used to configure tracing exporter.
	// +optional
	Tracing TracingConfig `json:"tracing"`

	// Disable webhooks.
	// +optional
	WebhooksDisabled bool `json:"webhooksDisabled"`
}

func init() {
	SchemeBuilder.Register(&ProjectConfig{})
}
