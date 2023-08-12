package v1

// Important: Run "make" to regenerate code after modifying this file

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkSpec defines the desired state of Work
type WorkSpec struct {
	// The [Cron](https://en.wikipedia.org/wiki/Cron) schedule for running the Work.
	// If not specified, the Work will be run as a one-shot Job.
	Schedule string `json:"schedule,omitempty"`

	// Override command when starting Docker image.
	Command []string `json:"command,omitempty"`

	// Your jobs's Docker image location and tag.
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
	// The ConfigMap and Secret resources must live in the same Kubernetes namespace as the Work resource.
	EnvFrom []EnvFrom `json:"envFrom,omitempty"`

	// RestartPolicy describes how the container should be restarted. Only one of the following restart policies may be specified.
	// If none of the following policies is specified, the default one is Never.
	// Read more about [Kubernetes handling pod and container failures](https://kubernetes.io/docs/concepts/workloads/controllers/job/#handling-pod-and-container-failures)
	// +kubebuilder:validation:Enum=OnFailure;Never
	RestartPolicy string `json:"restartPolicy,omitempty"`
}

type WorkStatus struct {
	// +optional
	Hash string `json:"hash,omitempty"`
	// +optional
	Extensions []string `json:"extensions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Work is the base Schema for the work API.
// This struct contains the base spec without any extensions.
//
// Fields that are not part of the base schema are stored in the `Rest` field.
type Work struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkSpec   `json:"spec,omitempty"`
	Status WorkStatus `json:"status,omitempty"`
}

func (w *Work) GetSpec() []byte {
	if w == nil {
		return nil
	}

	b, _ := json.Marshal(w.Spec)
	return b
}

func init() {
	// SchemeBuilder.Register(&Work{}, &WorkList{})
}
