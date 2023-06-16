package v1

import (
	"encoding/json"

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

func (r *Extension) validateExtension() error {
	errs := r.validateSpec()
	if len(errs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(r.GroupVersionKind().GroupKind(), r.Name, errs)
}

func (r *Extension) validateSpec() (allErrs field.ErrorList) {
	validations := []func() *field.Error{
		r.validateSpecTarget,
		r.validateSpecOpenAPIV3Schema,
	}

	for _, v := range validations {
		if err := v(); err != nil {
			allErrs = append(allErrs, err)
		}
	}

	return allErrs
}

func (r *Extension) validateSpecTarget() *field.Error {
	for _, target := range r.Spec.Targets {
		switch target {
		case "Application", "Work":
		default:
			return field.Invalid(field.NewPath("spec", "Targets"), target, "Must be one of [Application,Work]")
		}
	}
	return nil
}

func (r *Extension) validateSpecOpenAPIV3Schema() *field.Error {
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
