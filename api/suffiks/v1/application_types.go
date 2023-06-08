package v1

// Important: Run "make" to regenerate code after modifying this file

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/mitchellh/hashstructure/v2"
	"github.com/perimeterx/marshmallow"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ResourceRequirementsLimits struct {
	// Number of bytes to limit.
	// +kubebuilder:validation:Required
	Memory resource.Quantity `json:"memory"`
}

type ResourceRequirementsRequests struct {
	// Number of CPU units to request.
	// +kubebuilder:validation:Required
	CPU resource.Quantity `json:"cpu"`

	// Number of bytes to request.
	// +kubebuilder:validation:Required
	Memory resource.Quantity `json:"memory"`
}

type ResourceRequirements struct {
	// +kubebuilder:validation:Required
	Limits ResourceRequirementsLimits `json:"limits"`
	// +kubebuilder:validation:Required
	Requests ResourceRequirementsRequests `json:"requests"`
}

type EnvVars []EnvVar

type EnvVar struct {
	// Environment variable name. May only contain letters, digits, and the underscore `_` character.
	Name string `json:"name"`
	// Environment variable value. Numbers and boolean values must be quoted.
	// Required unless `valueFrom` is specified.
	Value string `json:"value,omitempty"`
	// Dynamically set environment variables based on fields found in the Pod spec.
	ValueFrom *EnvVarSource `json:"valueFrom,omitempty"`
}

type EnvVarSource struct {
	FieldRef ObjectFieldSelector `json:"fieldRef"`
}

type ObjectFieldSelector struct {
	// Field value from the `Pod` spec that should be copied into the environment variable.
	FieldPath string `json:"fieldPath" enum:";metadata.name;metadata.namespace;metadata.labels;metadata.annotations;spec.nodeName;spec.serviceAccountName;status.hostIP;status.podIP"`
}

type EnvFrom struct {
	// Name of the `ConfigMap` where environment variables are specified.
	// Required unless `secret` is set.
	ConfigMap string `json:"configmap,omitempty"`
	// Name of the `Secret` where environment variables are specified.
	// Required unless `configMap` is set.
	Secret string `json:"secret,omitempty"`
}

// ApplicationSpec defines the desired state of Application
type ApplicationSpec struct {
	// The port number which is exposed by the container and should receive traffic.
	Port int `json:"port,omitempty"`

	// Override command when starting Docker image.
	Command []string `json:"command,omitempty"`

	// Your application's Docker image location and tag.
	Image string `json:"image"`

	// Custom environment variables injected into your container.
	// Specify either `value` or `valueFrom`, but not both.
	Env EnvVars `json:"env,omitempty"`

	// EnvFrom exposes all variables in the ConfigMap or Secret resources as environment variables.
	// One of `configMap` or `secret` is required.
	//
	// Environment variables will take the form `KEY=VALUE`, where `key` is the ConfigMap or Secret key.
	// You can specify as many keys as you like in a single ConfigMap or Secret.
	//
	// The ConfigMap and Secret resources must live in the same Kubernetes namespace as the Application resource.
	EnvFrom []EnvFrom `json:"envFrom,omitempty"`

	//+optional
	Resources *ResourceRequirements `json:"resources,omitempty"`

	Rest unstructured.Unstructured `json:"-"`
}

type ApplicationStatus struct {
	// +optional
	AvailableReplicas int32 `json:"availableReplicas,omitempty"`
	// +optional
	Replicas int32 `json:"replicas,omitempty"`
	// +optional
	Hash string `json:"hash,omitempty"`
	// +optional
	Extensions []string `json:"extensions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=app
//+kubebuilder:object:root=true
// +genclient

// Application is the base Schema for the application API
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationSpec   `json:"spec,omitempty"`
	Status ApplicationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ExtensionList contains a list of Extension
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Application `json:"items"`
}

// Well known

func (a *Application) GetSpec() []byte {
	if a == nil {
		return nil
	}

	b, err := json.Marshal(a.Spec)
	if err != nil {
		panic(err)
	}
	return b
}

func (a *Application) WellKnownSpec() (ApplicationSpec, error) {
	if a == nil {
		return ApplicationSpec{}, nil
	}

	return a.Spec, nil
}

func (a *Application) Hash() (string, error) {
	if a == nil {
		return "", fmt.Errorf("unable to hash nil application")
	}

	v := struct {
		Spec   ApplicationSpec
		Labels map[string]string
	}{
		Spec:   a.Spec,
		Labels: a.Labels,
	}
	h, err := hashstructure.Hash(v, hashstructure.FormatV2, &hashstructure.HashOptions{
		IgnoreZeroValue: true,
	})
	if err != nil {
		return "", err
	}

	return strconv.FormatUint(h, 16), nil
}

func init() {
	SchemeBuilder.Register(&Application{}, &ApplicationList{})
}

type _app ApplicationSpec

func (a *ApplicationSpec) UnmarshalJSON(b []byte) error {
	var app _app
	rest, err := marshmallow.Unmarshal(b, &app, marshmallow.WithExcludeKnownFieldsFromMap(true))
	if err != nil {
		return err
	}

	*a = ApplicationSpec(app)
	a.Rest.Object = rest
	return nil
}

func (a ApplicationSpec) MarshalJSON() ([]byte, error) {
	b1, err := json.Marshal(_app(a))
	if err != nil {
		return nil, err
	}

	if len(a.Rest.Object) == 0 {
		return b1, nil
	}

	b2, err := json.Marshal(a.Rest.Object)
	if err != nil {
		return nil, err
	}

	b := append(append(b1[:len(b1)-1], ','), b2[1:]...)
	return b, nil
}
