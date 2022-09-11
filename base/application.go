// +kubebuilder:skip
package base

// import (
// 	"encoding/json"
// 	"fmt"
// 	"strconv"

// 	"github.com/mitchellh/hashstructure/v2"
// 	suffiksv1 "github.com/suffiks/suffiks/apis/suffiks/v1"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/apimachinery/pkg/runtime"
// )

// // +kubebuilder:object:generate=true
// // +kubebuilder:object:root=true

// // Application is the base Schema for the application API
// type Application struct {
// 	metav1.TypeMeta   `json:",inline"`
// 	metav1.ObjectMeta `json:"metadata,omitempty"`

// 	Spec   runtime.RawExtension        `json:"spec,omitempty"`
// 	Status suffiksv1.ApplicationStatus `json:"status,omitempty"`
// }

// func (a *Application) GetSpec() []byte {
// 	if a == nil {
// 		return nil
// 	}
// 	return a.Spec.Raw
// }

// func (a *Application) WellKnownSpec() (suffiksv1.ApplicationSpec, error) {
// 	if a == nil {
// 		return suffiksv1.ApplicationSpec{}, nil
// 	}
// 	spec := suffiksv1.ApplicationSpec{}
// 	err := json.Unmarshal(a.Spec.Raw, &spec)
// 	return spec, err
// }

// func (a *Application) Hash() (string, error) {
// 	if a == nil {
// 		return "", fmt.Errorf("unable to hash nil application")
// 	}

// 	v := struct {
// 		Spec   []byte
// 		Labels map[string]string
// 	}{
// 		Spec:   a.Spec.Raw[:],
// 		Labels: a.Labels,
// 	}
// 	h, err := hashstructure.Hash(v, hashstructure.FormatV2, &hashstructure.HashOptions{
// 		IgnoreZeroValue: true,
// 	})
// 	if err != nil {
// 		return "", err
// 	}

// 	return strconv.FormatUint(h, 16), nil
// }

// //+kubebuilder:object:root=true

// // ApplicationList contains a list of Application
// type ApplicationList struct {
// 	metav1.TypeMeta `json:",inline"`
// 	metav1.ListMeta `json:"metadata,omitempty"`
// 	Items           []Application `json:"items"`
// }

// func init() {
// 	suffiksv1.SchemeBuilder.Register(&Application{}, &ApplicationList{})
// }
