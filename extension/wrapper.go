package extension

import (
	"context"

	"github.com/suffiks/suffiks/extension/protogen"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type Owner struct {
	owner *protogen.Owner
}

func (o Owner) Kind() string                   { return o.owner.Kind }
func (o Owner) Name() string                   { return o.owner.Name }
func (o Owner) Namespace() string              { return o.owner.Namespace }
func (o Owner) Labels() map[string]string      { return o.owner.Labels }
func (o Owner) Annotations() map[string]string { return o.owner.Annotations }

// OwnerReference returns a OwnerReference for the kind that initiated the request.
func (o Owner) OwnerReference() metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: o.owner.ApiVersion,
		Kind:       o.Kind(),
		Name:       o.Name(),
		UID:        types.UID(o.owner.Uid),
	}
}

// ObjectMeta returns the ObjectMeta for the kind that initiated the request.
// It sets the name and namespace equal to the owner, and copies labels and annotations, while setting the owner reference.
func (o Owner) ObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:        o.Name(),
		Namespace:   o.Namespace(),
		Labels:      o.Labels(),
		Annotations: o.Annotations(),
		OwnerReferences: []metav1.OwnerReference{
			o.OwnerReference(),
		},
	}
}

type ValidationErrors struct {
	Path   string
	Value  string
	Detail string
}

type ValidationType int

func (v ValidationType) String() string {
	switch v {
	case ValidationCreate:
		return "create"
	case ValidationUpdate:
		return "update"
	case ValidationDelete:
		return "delete"
	}
	return "unknown"
}

const (
	ValidationCreate = iota
	ValidationUpdate
	ValidationDelete
)

type Extension[Object any] interface {
	Sync(ctx context.Context, owner Owner, obj Object, resp *ResponseWriter) error
	Delete(ctx context.Context, owner Owner, obj Object) (protogen.DeleteResponse, error)
}

type ValidatableExtension[Object any] interface {
	Validate(ctx context.Context, typ ValidationType, owner Owner, newObject, oldObject Object) ([]ValidationErrors, error)
}

type DefaultableExtension[Object any] interface {
	Default(ctx context.Context, owner Owner, obj Object) (Object, error)
}
