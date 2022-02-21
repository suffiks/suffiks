package testutil

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/runtime"
)

func AppSpec(fields map[string]any) runtime.RawExtension {
	root := map[string]any{
		"image": "some-image",
		"port":  8080,
	}
	for k, v := range fields {
		root[k] = v
	}

	re := runtime.RawExtension{}
	var err error

	re.Raw, err = json.Marshal(root)

	if err != nil {
		panic(err)
	}

	return re
}
