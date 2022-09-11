package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch/v5"
	suffiksv1 "github.com/suffiks/suffiks/apis/suffiks/v1"
	"github.com/suffiks/suffiks/base"
	"github.com/suffiks/suffiks/base/tracing"
	"go.opentelemetry.io/otel/attribute"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ Reconciler[*base.Application] = &AppReconciler{}

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
}

func (a *AppReconciler) NewObject() *base.Application { return &base.Application{} }

func (a *AppReconciler) CreateOrUpdate(ctx context.Context, app *base.Application, changeset *base.Changeset) error {
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

	if err := a.modifyDeployment(depl, changeset); err != nil {
		span.RecordError(err)
		return fmt.Errorf("unable to modify Deployment: %w", err)
	}

	depl.Spec.Template.Annotations = mergeMaps(depl.Spec.Template.Annotations, depl.Annotations)

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			Labels:    make(map[string]string),
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt(spec.Port),
				},
			},
			Selector: map[string]string{
				"app": app.Name,
			},
		},
	}
	for k, v := range depl.Labels {
		svc.Labels[k] = v
	}

	if err := a.Client.Create(ctx, svc); err != nil {
		if errors.IsAlreadyExists(err) {
			span.SetAttributes(attribute.String("action", "update svc"))
			if err := a.Client.Update(ctx, svc); err != nil {
				span.RecordError(err)
				return fmt.Errorf("Reconcile update svc: %w", err)
			}
		} else {
			span.RecordError(err)
			return fmt.Errorf("Reconcile create svc: %w", err)
		}
	} else {
		span.SetAttributes(attribute.String("action", "create svc"))
	}

	if err := a.Client.Create(ctx, depl); err != nil {
		if errors.IsAlreadyExists(err) {
			span.SetAttributes(attribute.String("action", "update depl"))
			if err := a.Client.Update(ctx, depl); err != nil {
				span.RecordError(err)
				return fmt.Errorf("Reconcile update depl: %w", err)
			}
		} else {
			span.RecordError(err)
			return fmt.Errorf("Reconcile create depl: %w", err)
		}
	} else {
		span.SetAttributes(attribute.String("action", "create depl"))
	}
	return nil
}

func (a *AppReconciler) UpdateStatus(ctx context.Context, app *base.Application, extensions []string) error {
	hash, err := app.Hash()
	if err != nil {
		return fmt.Errorf("error hashing application: %w", err)
	}
	app.Status.Hash = hash
	app.Status.Extensions = extensions

	depl := &appsv1.Deployment{}
	if err := a.Client.Get(ctx, client.ObjectKeyFromObject(app), depl); err != nil {
		return fmt.Errorf("error getting deployment: %w", err)
	}
	app.Status.Replicas = depl.Status.AvailableReplicas
	app.Status.AvailableReplicas = depl.Status.AvailableReplicas
	return nil
}

func (a *AppReconciler) IsModified(ctx context.Context, app *base.Application) (bool, error) {
	h, err := app.Hash()
	if err != nil {
		return false, err
	}

	if app.Status.Hash == h {
		ok := client.ObjectKeyFromObject(app)
		if err := a.Client.Get(ctx, ok, &appsv1.Deployment{}); err != nil && errors.IsNotFound(err) {
			return true, nil
		} else if err != nil {
			return false, err
		}

		// TODO(thokra): We don't yet create services
		if err := a.Client.Get(ctx, ok, &corev1.Service{}); err != nil && errors.IsNotFound(err) {
			return true, nil
		} else if err != nil {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

func (a *AppReconciler) Delete(ctx context.Context, app *base.Application) error {
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

func (a *AppReconciler) Extensions(app *base.Application) []string {
	return app.Status.Extensions
}

func (a *AppReconciler) Owns() []client.Object {
	return []client.Object{
		&appsv1.Deployment{},
		&corev1.Service{},
	}
}

func (a *AppReconciler) newDeployment(app *base.Application, spec suffiksv1.ApplicationSpec) *appsv1.Deployment {
	labels := map[string]string{
		"app": app.Name,
	}

	return &appsv1.Deployment{
		ObjectMeta: a.objectMeta(app),
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(1),
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
							Name:    app.Name,
							Image:   spec.Image,
							Command: spec.Command,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: int32(spec.Port),
								},
							},
						},
					},
				},
			},
		},
	}
}

func (a *AppReconciler) objectMeta(app *base.Application) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      app.Name,
		Namespace: app.Namespace,
	}
}

func (r *AppReconciler) modifyDeployment(depl *appsv1.Deployment, changeset *base.Changeset) error {
	depl.Labels = mergeMaps(depl.Labels, changeset.Labels)
	depl.Annotations = mergeMaps(depl.Annotations, changeset.Annotations)
	depl.Spec.Template.Spec.Containers[0].Env = append(depl.Spec.Template.Spec.Containers[0].Env, changeset.Environment...)
	depl.Spec.Template.Spec.Containers[0].EnvFrom = append(depl.Spec.Template.Spec.Containers[0].EnvFrom, changeset.EnvFrom...)

	if len(changeset.MergePatch) > 0 {
		b, err := json.Marshal(depl)
		if err != nil {
			return fmt.Errorf("modifyApp unmarshal: %w", err)
		}
		out, err := jsonpatch.MergePatch(b, changeset.MergePatch)
		if err != nil {
			return fmt.Errorf("modifyApp mergePatch: %w", err)
		}
		err = json.Unmarshal(out, depl)
		if err != nil {
			return fmt.Errorf("modifyApp unmarshal: %w", err)
		}
	}
	return nil
}
