package v1

// Important: Run "make" to regenerate code after modifying this file

import (
	"net"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

type ExtensionController struct {
	Namespace string `json:"namespace"`
	Service   string `json:"service"`
	Port      int    `json:"port"`
}

func (e ExtensionController) Target() string {
	return net.JoinHostPort(e.Service+"."+e.Namespace, strconv.Itoa(e.Port))
}

// +kubebuilder:validation:Enum=Application;Work
type Target string

type ExtensionWebhooks struct {
	Validation bool `json:"validation,omitempty"`
	Defaulting bool `json:"defaulting,omitempty"`
}

type ExtensionSpec struct {
	Targets []Target `json:"targets"`
	// +kubebuilder:pruning:PreserveUnknownFields
	OpenAPIV3Schema runtime.RawExtension `json:"openAPIV3Schema"`

	Controller ExtensionController `json:"controller"`
	Webhooks   ExtensionWebhooks   `json:"webhooks,omitempty"`

	// Always call the extension, even if the extension schema isn't set
	Always bool `json:"always,omitempty"`
}

type ExtensionStatusText string

const (
	ExtensionStatusApplied ExtensionStatusText = "Active"
	ExtensionStatusInvalid ExtensionStatusText = "Invalid"
)

type ExtensionStatus struct {
	// +optional
	Status ExtensionStatusText `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster,shortName=ext
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`
//+kubebuilder:printcolumn:name="Always",type=boolean,JSONPath=`.spec.always`
//+kubebuilder:printcolumn:name="Validation",type=boolean,JSONPath=`.spec.webhooks.validation`
//+kubebuilder:printcolumn:name="Defaulting",type=boolean,JSONPath=`.spec.webhooks.defaulting`

// Extension is the Schema for the extensions API
type Extension struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExtensionSpec   `json:"spec,omitempty"`
	Status ExtensionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ExtensionList contains a list of Extension
type ExtensionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Extension `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Extension{}, &ExtensionList{})
}
