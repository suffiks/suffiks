package controller

import "sigs.k8s.io/controller-runtime/pkg/client"

type Object interface {
	client.Object

	GetSpec() []byte
}

func mergeMaps(maps ...map[string]string) map[string]string {
	ret := map[string]string{}
	for _, m := range maps {
		for k, v := range m {
			ret[k] = v
		}
	}
	return ret
}
