package v1

import (
	"context"
	"encoding/json"
	"time"

	"github.com/suffiks/suffiks/internal/extension/oci"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var extensionlog = logf.Log.WithName("extension-resource")

func (r *Extension) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-suffiks-com-v1-extension,mutating=false,failurePolicy=fail,sideEffects=None,groups=suffiks.com,resources=extensions,verbs=create;update,versions=v1,name=vextension.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Extension{}

func (r *Extension) validateExtension(opts ...valOpts) error {
	validateOpts := &validateOpts{
		ociGetter: oci.Get,
	}

	for _, opt := range opts {
		opt(validateOpts)
	}

	errs := r.validateSpec(validateOpts)
	if len(errs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(r.GroupVersionKind().GroupKind(), r.Name, errs)
}

func (r *Extension) validateSpec(opts *validateOpts) (allErrs field.ErrorList) {
	validations := []func(opts *validateOpts) *field.Error{
		r.validateSpecTarget,
		r.validateSpecOpenAPIV3Schema,
		r.validateWASIImage,
	}

	for _, v := range validations {
		if err := v(opts); err != nil {
			allErrs = append(allErrs, err)
		}
	}

	return allErrs
}

func (r *Extension) validateSpecTarget(opts *validateOpts) *field.Error {
	for _, target := range r.Spec.Targets {
		switch target {
		case "Application", "Work":
		default:
			return field.Invalid(field.NewPath("spec", "Targets"), target, "Must be one of [Application,Work]")
		}
	}
	return nil
}

func (r *Extension) validateSpecOpenAPIV3Schema(opts *validateOpts) *field.Error {
	fieldPath := field.NewPath("spec", "openAPIV3Schema")
	props := &apiextv1.JSONSchemaProps{}
	if err := json.Unmarshal(r.Spec.OpenAPIV3Schema.Raw, props); err != nil {
		return field.Invalid(fieldPath, string(r.Spec.OpenAPIV3Schema.Raw), "Must be a valid JSONSchema")
	}
	if props.Type != "object" {
		return field.Invalid(fieldPath, string(r.Spec.OpenAPIV3Schema.Raw), "Must be of type 'object'")
	}
	if len(props.Properties) == 0 && !r.Spec.Always {
		return field.Invalid(fieldPath, string(r.Spec.OpenAPIV3Schema.Raw), "Must have at least one property or have Always set to true")
	}
	return nil
}

func (r *Extension) validateWASIImage(opts *validateOpts) *field.Error {
	if r.Spec.Controller.WASI == nil {
		return nil
	}

	if r.Spec.Controller.WASI.Image == "" {
		return field.Invalid(field.NewPath("spec", "controller", "wasi", "image"), r.Spec.Controller.WASI.Image, "Must be a valid image")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := opts.ociGetter(ctx, r.Spec.Controller.WASI.Image, r.Spec.Controller.WASI.Tag)
	if err != nil {
		return field.Invalid(field.NewPath("spec", "controller", "wasi", "image"), r.Spec.Controller.WASI.Image, err.Error())
	}

	return nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Extension) ValidateCreate() (admission.Warnings, error) {
	extensionlog.Info("validate create", "name", r.Name)
	return nil, r.validateExtension()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Extension) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	extensionlog.Info("validate update", "name", r.Name)
	return nil, r.validateExtension()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Extension) ValidateDelete() (admission.Warnings, error) {
	// NOT IN USE
	return nil, nil
}

type (
	ociGetter    func(ctx context.Context, image string, tag string) (map[string][]byte, error)
	validateOpts struct {
		ociGetter ociGetter
	}
	valOpts func(*validateOpts)
)

func withOciGetter(getter ociGetter) func(*validateOpts) {
	return func(opts *validateOpts) {
		opts.ociGetter = getter
	}
}
