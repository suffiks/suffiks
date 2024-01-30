package controller

import "sigs.k8s.io/controller-runtime/pkg/client"

type Object interface {
	client.Object

	GetSpec() []byte
}

// mergeMaps merges two maps, overwriting the values of the first map with the values of the later map.
func mergeMaps(maps ...map[string]string) map[string]string {
	ret := map[string]string{}
	for _, m := range maps {
		for k, v := range m {
			ret[k] = v
		}
	}
	return ret
}
