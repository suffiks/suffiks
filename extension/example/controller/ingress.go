package controller

import (
	"context"

	"github.com/suffiks/suffiks/extension"
	"github.com/suffiks/suffiks/extension/protogen"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var _ extension.Extension[*Ingresses] = &IngressExtension{}

// +kubebuilder:validation:Pattern=`^[\w\.\-]+$`
type Host string

// +kubebuilder:validation:Pattern=`^\/[\w\.\/\-]+$`
type Path string

func (p Path) IngressPath(svc, portName string) netv1.HTTPIngressPath {
	return netv1.HTTPIngressPath{
		Path: string(p),
		Backend: netv1.IngressBackend{
			Service: &netv1.IngressServiceBackend{
				Name: svc,
				Port: netv1.ServiceBackendPort{
					Name: portName,
				},
			},
		},
	}
}

type Ingress struct {
	Host  Host   `json:"host"`
	Paths []Path `json:"paths,omitempty"`
}

func (i *Ingress) Rule(svc string) netv1.IngressRule {
	ir := netv1.IngressRule{
		Host: string(i.Host),
		IngressRuleValue: netv1.IngressRuleValue{
			HTTP: &netv1.HTTPIngressRuleValue{
				Paths: []netv1.HTTPIngressPath{},
			},
		},
	}

	for _, path := range i.Paths {
		ir.HTTP.Paths = append(ir.HTTP.Paths, path.IngressPath(svc, "http"))
	}

	if len(i.Paths) == 0 {
		ir.HTTP.Paths = []netv1.HTTPIngressPath{
			Path("/").IngressPath(svc, "http"),
		}
	}

	return ir
}

// +suffiks:extension:Targets=Application
type Ingresses struct {
	// List of URLs that will route HTTPS traffic to the application.
	// All URLs must start with `https://`. Domain availability differs according to which environment your application is running in.
	Ingresses []Ingress `json:"ingresses,omitempty"`
}

type IngressExtension struct {
	Client kubernetes.Interface
}

func (i *IngressExtension) Sync(ctx context.Context, owner extension.Owner, ingress *Ingresses, resp *extension.ResponseWriter) error {
	var rules []netv1.IngressRule
	for _, ingress := range ingress.Ingresses {
		rules = append(rules, ingress.Rule(owner.Name()))
	}

	ing, err := i.Client.NetworkingV1().Ingresses(owner.Namespace()).Get(ctx, owner.Name()+"-ing", metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		ing = &netv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      owner.Name() + "-ing",
				Namespace: owner.Namespace(),
				OwnerReferences: []metav1.OwnerReference{
					owner.OwnerReference(),
				},
			},
			Spec: netv1.IngressSpec{
				Rules: rules,
			},
		}

		_, err = i.Client.NetworkingV1().Ingresses(owner.Namespace()).Create(ctx, ing, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	} else {

		ing.Spec = netv1.IngressSpec{
			Rules: rules,
		}

		_, err = i.Client.NetworkingV1().Ingresses(owner.Namespace()).Update(ctx, ing, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *IngressExtension) Delete(ctx context.Context, owner extension.Owner, v *Ingresses) (protogen.DeleteResponse, error) {
	err := i.Client.NetworkingV1().Ingresses(owner.Namespace()).Delete(ctx, owner.Name()+"-ing", metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return protogen.DeleteResponse{
			Error: err.Error(),
		}, err
	}

	return protogen.DeleteResponse{}, nil
}
