package testutil

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func OwnerReference(kind, name string) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: "v1",
		Kind:       kind,
		Name:       name,
	}
}
