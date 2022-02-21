// +kubebuilder:skip
package base

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/mitchellh/hashstructure/v2"
	suffiksv1 "github.com/suffiks/suffiks/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// Work is the base Schema for the work API
type Work struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   runtime.RawExtension `json:"spec,omitempty"`
	Status suffiksv1.WorkStatus `json:"status,omitempty"`
}

func (n *Work) GetSpec() []byte {
	if n == nil {
		return nil
	}
	return n.Spec.Raw
}

func (n *Work) WellKnownSpec() (suffiksv1.WorkSpec, error) {
	if n == nil {
		return suffiksv1.WorkSpec{}, nil
	}
	spec := suffiksv1.WorkSpec{}
	err := json.Unmarshal(n.Spec.Raw, &spec)
	return spec, err
}

func (n *Work) Hash() (string, error) {
	if n == nil {
		return "", fmt.Errorf("unable to hash nil job")
	}

	v := struct {
		Spec   []byte
		Labels map[string]string
	}{
		Spec:   n.Spec.Raw[:],
		Labels: n.Labels,
	}
	h, err := hashstructure.Hash(v, hashstructure.FormatV2, &hashstructure.HashOptions{
		IgnoreZeroValue: true,
	})
	if err != nil {
		return "", err
	}

	return strconv.FormatUint(h, 16), nil
}

//+kubebuilder:object:root=true

// WorkList contains a list of Work
type WorkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Work `json:"items"`
}

func init() {
	suffiksv1.SchemeBuilder.Register(&Work{}, &WorkList{})
}
