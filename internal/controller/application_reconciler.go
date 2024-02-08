package controller

import (
	"context"
	"fmt"

	"github.com/suffiks/suffiks/internal/extension"
	"github.com/suffiks/suffiks/internal/tracing"
	suffiksv1 "github.com/suffiks/suffiks/pkg/api/suffiks/v1"
	"go.opentelemetry.io/otel/attribute"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"k8s.io/utils/strings/slices"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	_ Reconciler[*suffiksv1.Application]        = &AppReconciler{}
	_ ReconcilerDefault[*suffiksv1.Application] = &AppReconciler{}
)

// When changing the lines below, run make
//+kubebuilder:rbac:groups=suffiks.com,resources=applications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=suffiks.com,resources=applications/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=suffiks.com,resources=applications/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// ValidationWebhook
//+kubebuilder:webhook:path=/validate-suffiks-com-v1-application,mutating=false,failurePolicy=fail,sideEffects=None,groups=suffiks.com,resources=applications,verbs=create;update;delete,versions=v1,name=vapplication.kb.io,admissionReviewVersions=v1
// DefaultingWebhook
//+kubebuilder:webhook:path=/mutate-suffiks-com-v1-application,mutating=true,failurePolicy=fail,sideEffects=None,groups=suffiks.com,resources=applications,verbs=create;update,versions=v1,name=mapplication.kb.io,admissionReviewVersions=v1

type AppReconciler struct {
	Scheme *runtime.Scheme
	Client client.Client

	// Defaults *suffiksv1.ApplicationDefaults
}

func (a *AppReconciler) NewObject() *suffiksv1.Application { return &suffiksv1.Application{} }

func (a *AppReconciler) CreateOrUpdate(ctx context.Context, app *suffiksv1.Application, changeset *extension.Changeset) error {
	ctx, span := tracing.Start(ctx, "AppReconciler.CreateOrUpdate")
	defer span.End()

	spec, err := app.WellKnownSpec()
	if err != nil {
		return err
	}

	depl := a.newDeployment(app, spec)
	if err := controllerutil.SetControllerReference(app, depl, a.Scheme); err != nil {
		span.RecordError(err)
		return fmt.Errorf("unable to set controller reference: %w", err)
	}

	if err := changeset.Apply(depl); err != nil {
		span.RecordError(err)
		return fmt.Errorf("unable to modify Deployment: %w", err)
	}

	depl.Spec.Template.Annotations = mergeMaps(depl.Spec.Template.Annotations, depl.Annotations)

	if spec.Port > 0 {
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      app.Name,
				Namespace: app.Namespace,
			},
		}
		if err := a.Client.Get(ctx, client.ObjectKeyFromObject(svc), svc); err != nil && !errors.IsNotFound(err) {
			span.RecordError(err)
			return fmt.Errorf("error getting service: %w", err)
		}

		update := err == nil || !errors.IsNotFound(err)
		svc.Spec = corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt(spec.Port),
				},
			},
			Selector: map[string]string{
				"app.kubernetes.io/name": app.Name,
			},
		}
		if svc.Labels == nil {
			svc.Labels = map[string]string{}
		}
		for k, v := range depl.Labels {
			svc.Labels[k] = v
		}

		if update {
			span.SetAttributes(attribute.String("action", "update svc"))
			if err := a.Client.Update(ctx, svc); err != nil {
				span.RecordError(err)
				return fmt.Errorf("Reconcile update svc: %w", err)
			}
		} else {
			span.SetAttributes(attribute.String("action", "create svc"))
			if err := a.Client.Create(ctx, svc); err != nil {
				span.RecordError(err)
				return fmt.Errorf("Reconcile create svc: %w", err)
			}
		}
	}

	existingDepl := &appsv1.Deployment{}
	if err := a.Client.Get(ctx, client.ObjectKeyFromObject(depl), existingDepl); err != nil && !errors.IsNotFound(err) {
		span.RecordError(err)
		return fmt.Errorf("error getting deployment: %w", err)
	}

	if err == nil && existingDepl.Generation > 0 {
		depl.Generation = existingDepl.Generation
		span.SetAttributes(attribute.String("action", "update deployment"))
		if err := a.Client.Update(ctx, depl); err != nil {
			span.RecordError(err)
			return fmt.Errorf("Reconcile update deployment: %w", err)
		}
	} else {
		if err := a.Client.Create(ctx, depl); err != nil {
			span.RecordError(err)
			return fmt.Errorf("Reconcile create deployment: %w", err)
		}
	}
	return nil
}

func (a *AppReconciler) UpdateStatus(ctx context.Context, app *suffiksv1.Application, extensions []string) (updates bool, err error) {
	hash, err := app.Hash()
	if err != nil {
		return updates, fmt.Errorf("error hashing application: %w", err)
	}
	if app.Status.Hash != hash {
		updates = true
		app.Status.Hash = hash
	}

	if !slices.Equal(app.Status.Extensions, extensions) {
		updates = true
		app.Status.Extensions = extensions
	}

	depl := &appsv1.Deployment{}
	if err := a.Client.Get(ctx, client.ObjectKeyFromObject(app), depl); err != nil {
		return updates, nil
		// return updates, fmt.Errorf("error getting deployment: %w", err)
	}

	if app.Status.Replicas != depl.Status.AvailableReplicas {
		updates = true
		app.Status.Replicas = depl.Status.AvailableReplicas
	}

	if app.Status.AvailableReplicas != depl.Status.AvailableReplicas {
		updates = true
		app.Status.AvailableReplicas = depl.Status.AvailableReplicas
	}
	return updates, nil
}

func (a *AppReconciler) IsModified(ctx context.Context, app *suffiksv1.Application) (bool, error) {
	h, err := app.Hash()
	if err != nil {
		tracing.Get(ctx).RecordError(fmt.Errorf("IsModified: get app hash: %w", err))
		return false, err
	}

	if app.Status.Hash != h {
		tracing.Get(ctx).AddEvent("Hash mismatch")
		return true, nil
	}

	ok := client.ObjectKeyFromObject(app)
	if err := a.Client.Get(ctx, ok, &appsv1.Deployment{}); err != nil && errors.IsNotFound(err) {
		return true, nil
	} else if err != nil {
		return false, err
	}
	tracing.Get(ctx).AddEvent("Got from client")

	spec, err := app.WellKnownSpec()
	if err != nil {
		tracing.Get(ctx).RecordError(fmt.Errorf("IsModified: get well known spec: %w", err))
		return false, err
	}
	tracing.Get(ctx).AddEvent("Got well known spec")

	if spec.Port > 0 {
		tracing.Get(ctx).AddEvent("Checking service")
		if err := a.Client.Get(ctx, ok, &corev1.Service{}); err != nil && errors.IsNotFound(err) {
			return true, nil
		} else if err != nil {
			tracing.Get(ctx).RecordError(fmt.Errorf("IsModified: get service: %w", err))
			return false, err
		}
		tracing.Get(ctx).AddEvent("Done checking service")
	}
	return false, nil
}

func (a *AppReconciler) Delete(ctx context.Context, app *suffiksv1.Application) error {
	err := a.Client.Delete(ctx, &appsv1.Deployment{ObjectMeta: a.objectMeta(app)})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	err = a.Client.Delete(ctx, &corev1.Service{ObjectMeta: a.objectMeta(app)})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

func (a *AppReconciler) Extensions(app *suffiksv1.Application) []string {
	return app.Status.Extensions
}

func (a *AppReconciler) Owns() []client.Object {
	return []client.Object{
		// TODO: Fix recursive reconciliation loop before enabling this
		// &appsv1.Deployment{},
		// &corev1.Service{},
	}
}

func (a *AppReconciler) Default(ctx context.Context, obj *suffiksv1.Application) error {
	// if obj.Spec.Resources == nil && a.Defaults != nil && a.Defaults.Resources != nil {
	// 	span := tracing.Get(ctx)
	// 	span.AddEvent("add_default_resources")
	// 	obj.Spec.Resources = a.Defaults.Resources.DeepCopy()
	// }
	return nil
}

func (a *AppReconciler) newDeployment(app *suffiksv1.Application, spec suffiksv1.ApplicationSpec) *appsv1.Deployment {
	labels := map[string]string{
		"app.kubernetes.io/name": app.Name,
	}

	ports := []corev1.ContainerPort{}
	if spec.Port > 0 {
		ports = append(ports, corev1.ContainerPort{
			Name:          "http",
			ContainerPort: int32(spec.Port),
		})
	}

	rq := corev1.ResourceRequirements{}
	if spec.Resources != nil {
		rq = corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: spec.Resources.Limits.Memory,
			},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: spec.Resources.Requests.Memory,
				corev1.ResourceCPU:    spec.Resources.Requests.CPU,
			},
		}
	}

	return &appsv1.Deployment{
		ObjectMeta: a.objectMeta(app),
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To[int32](1),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:      app.Name,
							Image:     spec.Image,
							Command:   spec.Command,
							Ports:     ports,
							Resources: rq,
							Env:       envVars(spec.Env),
							EnvFrom:   envFroms(spec.EnvFrom),
						},
					},
				},
			},
		},
	}
}

func envFroms(froms []suffiksv1.EnvFrom) []corev1.EnvFromSource {
	result := make([]corev1.EnvFromSource, 0, len(froms))
	for _, from := range froms {
		if len(from.ConfigMap) > 0 {
			result = append(result, corev1.EnvFromSource{
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: from.ConfigMap,
					},
					Optional: ptr.To(true),
				},
			})
		} else {
			result = append(result, corev1.EnvFromSource{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: from.Secret,
					},
					Optional: ptr.To(true),
				},
			})
		}
	}
	return result
}

func envVars(vars suffiksv1.EnvVars) []corev1.EnvVar {
	result := make([]corev1.EnvVar, 0, len(vars))
	for _, env := range vars {
		if len(env.Value) > 0 {
			result = append(result, corev1.EnvVar{
				Name:  env.Name,
				Value: env.Value,
			})
		} else {
			result = append(result, corev1.EnvVar{
				Name: env.Name,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: env.ValueFrom.FieldRef.FieldPath,
					},
				},
			})
		}
	}

	return result
}

func (a *AppReconciler) objectMeta(app *suffiksv1.Application) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      app.Name,
		Namespace: app.Namespace,
	}
}
