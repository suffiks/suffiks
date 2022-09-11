package testutil

import (
	"encoding/json"

	suffiksv1 "github.com/suffiks/suffiks/apis/suffiks/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func AppSpec(wellKnown *suffiksv1.ApplicationSpec, fields map[string]any) suffiksv1.ApplicationSpec {
	if wellKnown == nil {
		wellKnown = &suffiksv1.ApplicationSpec{
			Image: "some-image",
			Port:  8080,
		}
	}

	re := runtime.RawExtension{}
	var err error

	re.Raw, err = json.Marshal(fields)
	if err != nil {
		panic(err)
	}

	wellKnown.RawExtension = re

	return *wellKnown
}
