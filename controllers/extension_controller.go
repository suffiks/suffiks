package controllers

import (
	"context"
	goerrors "errors"

	suffiksv1 "github.com/suffiks/suffiks/api/v1"
	"github.com/suffiks/suffiks/base"
	suffiksruntime "github.com/suffiks/suffiks/base/runtime"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type crdDefinition struct {
	Kind       suffiksv1.Target
	createFunc func(name string, schema *apiextv1.JSONSchemaProps) *apiextv1.CustomResourceDefinition
}

var crds = map[string]crdDefinition{
	"applications.suffiks.com": {
		Kind:       "Application",
		createFunc: createAppCRD,
	},
	"works.suffiks.com": {
		Kind:       "Work",
		createFunc: createWorkCRD,
	},
}

// ExtensionReconciler reconciles a Extension object
type ExtensionReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	CRDManager *base.ExtensionManager
	KubeConfig *rest.Config

	clientSet apiclient.Interface
}

//+kubebuilder:rbac:groups=suffiks.com,resources=extensions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=suffiks.com,resources=extensions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=suffiks.com,resources=extensions/finalizers,verbs=update
//+kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ExtensionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	ext := &suffiksv1.Extension{}
	if err := r.Get(ctx, req.NamespacedName, ext); err != nil {
		log.Error(err, "unable to fetch Extension")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	const finalizer = "extensions.suffiks.com/finalizer"
	if ext.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(ext, finalizer) {
			if err := r.CRDManager.Add(*(ext.DeepCopy())); err != nil {
				if goerrors.Is(err, &suffiksruntime.AlreadyDefinedError{}) {
					log.Info("CRD already exists, skipping", "error", err)
				} else {
					log.Error(err, "unable to add extension manifest")
				}
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return ctrl.Result{}, err
			}
			controllerutil.AddFinalizer(ext, finalizer)
			if err := r.Update(ctx, ext); err != nil {
				log.Error(err, "unable to update Extension")
				return ctrl.Result{}, err
			}

			for name, def := range crds {
				crd, err := r.clientSet.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, name, v1.GetOptions{})
				if err != nil {
					log.Error(err, "fetch crd"+name)
					return ctrl.Result{}, err
				}

				crd.Spec.Versions[0].Schema.OpenAPIV3Schema = r.CRDManager.Schema(def.Kind)
				_, err = r.clientSet.ApiextensionsV1().CustomResourceDefinitions().Update(ctx, crd, v1.UpdateOptions{})
				if err != nil && !errors.IsNotFound(err) {
					log.Error(err, "unable to update CRD")
					return ctrl.Result{}, err
				}
			}
			ext.Status.Status = suffiksv1.ExtensionStatusApplied
			if err := r.Status().Update(ctx, ext); err != nil {
				log.Error(err, "unable to update Extension status")
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if !controllerutil.ContainsFinalizer(ext, finalizer) {
			// Stop reconciliation as the item is being deleted
			return ctrl.Result{}, nil
		}

		// our finalizer is present, so lets handle any external dependency
		if err := r.CRDManager.Remove(ext); err != nil {
			// if fail to delete the external dependency here, return with error
			// so that it can be retried
			return ctrl.Result{}, err
		}

		// remove our finalizer from the list and update it.
		controllerutil.RemoveFinalizer(ext, finalizer)
		if err := r.Update(ctx, ext); err != nil {
			return ctrl.Result{}, err
		}

	}

	return ctrl.Result{}, nil
}

func (r *ExtensionReconciler) RefreshCRD(ctx context.Context) error {
	client, err := client.New(r.KubeConfig, client.Options{Scheme: r.Scheme})
	if err != nil {
		return err
	}

	list := &suffiksv1.ExtensionList{}
	if err := client.List(ctx, list); err != nil {
		return err
	}

	for _, ext := range list.Items {
		if err := r.CRDManager.Add(*(ext.DeepCopy())); err != nil {
			return err
		}
	}

	for name, def := range crds {
		crd, err := r.clientSet.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, name, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				crd = def.createFunc(name, r.CRDManager.Schema(def.Kind))
				_, err = r.clientSet.ApiextensionsV1().CustomResourceDefinitions().Create(ctx, crd, v1.CreateOptions{})
				if err != nil {
					return err
				}
			} else {
				return err
			}
			continue
		}

		// It exists, we can update it
		crd.Spec.Versions[0].Schema.OpenAPIV3Schema = r.CRDManager.Schema(def.Kind)
		_, err = r.clientSet.ApiextensionsV1().CustomResourceDefinitions().Update(ctx, crd, v1.UpdateOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ExtensionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.clientSet = apiclient.NewForConfigOrDie(r.KubeConfig)
	return ctrl.NewControllerManagedBy(mgr).
		For(&suffiksv1.Extension{}).
		Complete(r)
}

func createAppCRD(name string, schema *apiextv1.JSONSchemaProps) *apiextv1.CustomResourceDefinition {
	return &apiextv1.CustomResourceDefinition{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		Spec: apiextv1.CustomResourceDefinitionSpec{
			Group: suffiksv1.GroupVersion.Group,
			Scope: apiextv1.NamespaceScoped,
			Names: apiextv1.CustomResourceDefinitionNames{
				Kind:     "Application",
				ListKind: "ApplicationList",
				Singular: "application",
				Plural:   "applications",
				ShortNames: []string{
					"app",
					"apps",
				},
			},
			Versions: []apiextv1.CustomResourceDefinitionVersion{
				{
					Name:    "v1",
					Served:  true,
					Storage: true,
					Subresources: &apiextv1.CustomResourceSubresources{
						Status: &apiextv1.CustomResourceSubresourceStatus{},
					},
					Schema: &apiextv1.CustomResourceValidation{
						OpenAPIV3Schema: schema,
					},
				},
			},
		},
	}
}

func createWorkCRD(name string, schema *apiextv1.JSONSchemaProps) *apiextv1.CustomResourceDefinition {
	return &apiextv1.CustomResourceDefinition{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		Spec: apiextv1.CustomResourceDefinitionSpec{
			Group: suffiksv1.GroupVersion.Group,
			Scope: apiextv1.NamespaceScoped,
			Names: apiextv1.CustomResourceDefinitionNames{
				Kind:     "Work",
				ListKind: "WorkList",
				Singular: "work",
				Plural:   "works",
				ShortNames: []string{
					"nj",
				},
			},
			Versions: []apiextv1.CustomResourceDefinitionVersion{
				{
					Name:    "v1",
					Served:  true,
					Storage: true,
					Subresources: &apiextv1.CustomResourceSubresources{
						Status: &apiextv1.CustomResourceSubresourceStatus{},
					},
					Schema: &apiextv1.CustomResourceValidation{
						OpenAPIV3Schema: schema,
					},
				},
			},
		},
	}
}
