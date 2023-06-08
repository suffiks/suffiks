package controller

// import (
// 	"context"
// 	"fmt"

// 	suffiksv1 "github.com/suffiks/suffiks/api/suffiks/v1"
// 	"github.com/suffiks/suffiks/base"
// 	"github.com/suffiks/suffiks/base/tracing"
// 	"go.opentelemetry.io/otel/trace"
// 	batchv1 "k8s.io/api/batch/v1"
// 	corev1 "k8s.io/api/core/v1"
// 	"k8s.io/apimachinery/pkg/api/errors"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/apimachinery/pkg/runtime"
// 	"sigs.k8s.io/controller-runtime/pkg/client"
// 	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
// )

// var _ = &ReconcilerWrapper[*base.Work]{
// 	Child: &JobReconciler{},
// }

// // When changing the lines below, run make
// //+kubebuilder:rbac:groups=suffiks.com,resources=works,verbs=get;list;watch;create;update;patch;delete
// //+kubebuilder:rbac:groups=suffiks.com,resources=works/status,verbs=get;update;patch
// //+kubebuilder:rbac:groups=suffiks.com,resources=works/finalizers,verbs=update
// //+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// //+kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete

// // ValidationWebhook
// //+kubebuilder:webhook:path=/validate-suffiks-com-v1-work,mutating=false,failurePolicy=fail,sideEffects=None,groups=suffiks.com,resources=works,verbs=create;update;delete,versions=v1,name=vwork.kb.io,admissionReviewVersions=v1
// // DefaultingWebhook
// //+kubebuilder:webhook:path=/mutate-suffiks-com-v1-work,mutating=true,failurePolicy=fail,sideEffects=None,groups=suffiks.com,resources=works,verbs=create;update,versions=v1,name=mwork.kb.io,admissionReviewVersions=v1

// // JobReconciler reconciles a Work object
// type JobReconciler struct {
// 	Client client.Client
// 	Scheme *runtime.Scheme

// 	// KubeConfig    *rest.Config
// 	// CRDController *base.ExtensionController

// 	// klient kubernetes.Interface
// 	// rest   *rest.RESTClient
// }

// func (j *JobReconciler) NewObject() *base.Work { return &base.Work{} }
// func (j *JobReconciler) Owns() []client.Object { return nil }

// func (j *JobReconciler) CreateOrUpdate(ctx context.Context, job *base.Work, changeset *base.Changeset) error {
// 	ctx, span := tracing.Start(ctx, "AppReconciler.CreateOrUpdate")
// 	defer span.End()

// 	spec, err := job.WellKnownSpec()
// 	if err != nil {
// 		return fmt.Errorf("CreateOrUpdate: %w", err)
// 	}

// 	if spec.Schedule == "" {
// 		err = j.createOrUpdateJob(ctx, job, spec, changeset)
// 	} else {
// 		err = j.createOrUpdateCronJob(ctx, job, spec, changeset)
// 	}

// 	return err
// }

// func (j *JobReconciler) Delete(ctx context.Context, job *base.Work) error {
// 	return j.Client.Delete(ctx, &batchv1.Job{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      job.Name,
// 			Namespace: job.Namespace,
// 		},
// 	})
// }

// func (j *JobReconciler) createOrUpdateJob(ctx context.Context, job *base.Work, spec suffiksv1.WorkSpec, changeset *base.Changeset) error {
// 	jb := &batchv1.Job{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      job.Name,
// 			Namespace: job.Namespace,
// 			Labels:    job.Labels,
// 		},
// 		Spec: j.jobSpec(job, spec, changeset),
// 	}

// 	if err := controllerutil.SetControllerReference(job, jb, j.Scheme); err != nil {
// 		trace.SpanFromContext(ctx).RecordError(err)
// 		return fmt.Errorf("unable to set controller reference: %w", err)
// 	}

// 	if err := j.Client.Create(ctx, jb); err != nil {
// 		if !errors.IsAlreadyExists(err) {
// 			// If it already exists, then we don't need to do anything.
// 			return fmt.Errorf("Reconcile create: %w", err)
// 		}
// 	}
// 	return nil
// }

// func (j *JobReconciler) createOrUpdateCronJob(ctx context.Context, job *base.Work, spec suffiksv1.WorkSpec, changeset *base.Changeset) error {
// 	jb := &batchv1.CronJob{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      job.Name,
// 			Namespace: job.Namespace,
// 			Labels:    job.Labels,
// 		},
// 		Spec: batchv1.CronJobSpec{
// 			Schedule: spec.Schedule,
// 			JobTemplate: batchv1.JobTemplateSpec{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Labels: job.Labels,
// 				},
// 				Spec: j.jobSpec(job, spec, changeset),
// 			},
// 		},
// 	}

// 	if err := controllerutil.SetControllerReference(job, jb, j.Scheme); err != nil {
// 		trace.SpanFromContext(ctx).RecordError(err)
// 		return fmt.Errorf("unable to set controller reference: %w", err)
// 	}

// 	if err := j.Client.Create(ctx, jb); err != nil {
// 		if errors.IsAlreadyExists(err) {
// 			if err := j.Client.Update(ctx, jb); err != nil {
// 				return fmt.Errorf("Reconcile update: %w", err)
// 			}
// 		} else {
// 			return fmt.Errorf("Reconcile create: %w", err)
// 		}
// 	}
// 	return nil
// }

// func (a *JobReconciler) UpdateStatus(ctx context.Context, job *base.Work, extensions []string) error {
// 	hash, err := job.Hash()
// 	if err != nil {
// 		return fmt.Errorf("error hashing application: %w", err)
// 	}
// 	job.Status.Extensions = extensions
// 	job.Status.Hash = hash
// 	return nil
// }

// func (j *JobReconciler) IsModified(ctx context.Context, job *base.Work) (bool, error) {
// 	h, err := job.Hash()
// 	if err != nil {
// 		return false, err
// 	}

// 	if job.Status.Hash == h {
// 		// ok := client.ObjectKeyFromObject(job)
// 		// if err := j.Client.Get(ctx, ok, &appsv1.Deployment{}); err != nil && errors.IsNotFound(err) {
// 		// 	return true, nil
// 		// } else if err != nil {
// 		// 	return false, err
// 		// }
// 		return false, nil
// 	}
// 	return true, nil
// }

// func (j *JobReconciler) Extensions(job *base.Work) []string {
// 	return job.Status.Extensions
// }

// func (r *JobReconciler) jobSpec(job *base.Work, spec suffiksv1.WorkSpec, changeset *base.Changeset) batchv1.JobSpec {
// 	container := r.newContainer(job, spec, changeset)

// 	return batchv1.JobSpec{
// 		Template: corev1.PodTemplateSpec{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Labels: job.Labels,
// 			},
// 			Spec: corev1.PodSpec{
// 				RestartPolicy: corev1.RestartPolicy(spec.RestartPolicy),
// 				Containers: []corev1.Container{
// 					container,
// 				},
// 			},
// 		},
// 	}
// }

// func (r *JobReconciler) newContainer(work *base.Work, spec suffiksv1.WorkSpec, changeset *base.Changeset) corev1.Container {
// 	return corev1.Container{
// 		Name:            work.Name,
// 		Image:           spec.Image,
// 		Command:         spec.Command,
// 		ImagePullPolicy: corev1.PullIfNotPresent,
// 		Env:             changeset.Environment,
// 		EnvFrom:         changeset.EnvFrom,
// 	}
// }
