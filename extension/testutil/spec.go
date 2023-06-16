package testutil

import (
	suffiksv1 "github.com/suffiks/suffiks/pkg/api/suffiks/v1"
)

func AppSpec(wellKnown *suffiksv1.ApplicationSpec, fields map[string]any) suffiksv1.ApplicationSpec {
	if wellKnown == nil {
		wellKnown = &suffiksv1.ApplicationSpec{
			Image: "some-image",
			Port:  8080,
		}
	}

	wellKnown.Rest.Object = fields
	return *wellKnown
}
