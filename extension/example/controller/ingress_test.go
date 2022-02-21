package controller

import (
	"os"
	"testing"

	"github.com/suffiks/suffiks/base"
	"github.com/suffiks/suffiks/extension"
	"github.com/suffiks/suffiks/extension/testutil"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func newIngressExtension(client *fake.Clientset) extension.Extension[*Ingresses] {
	return &IngressExtension{
		Client: client,
	}
}

func TestIntegration(t *testing.T) {
	app := &base.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "some-app",
			Namespace: "mynamespace",
		},
		Spec: testutil.AppSpec(map[string]any{
			"ingresses": []map[string]any{
				{
					"host": "mydomain.org",
					"paths": []string{
						"/some/path",
					},
				},
			},
		}),
	}

	tests := []testutil.TestCase{
		testutil.SyncTest{
			Name:   "create application",
			Object: app,
			Expected: []runtime.Object{
				&netv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "some-app-ing",
						Namespace: "mynamespace",
						OwnerReferences: []metav1.OwnerReference{
							testutil.OwnerReference("Application", "some-app"),
						},
					},
					Spec: netv1.IngressSpec{
						Rules: []netv1.IngressRule{
							{
								Host: "mydomain.org",
								IngressRuleValue: netv1.IngressRuleValue{
									HTTP: &netv1.HTTPIngressRuleValue{
										Paths: []netv1.HTTPIngressPath{
											{
												Path: "/some/path",
												Backend: netv1.IngressBackend{
													Service: &netv1.IngressServiceBackend{
														Name: "some-app",
														Port: netv1.ServiceBackendPort{
															Name: "http",
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},

		testutil.DeleteTest{
			Name:   "delete application",
			Object: app,
			ExpectedDeleted: []testutil.Deleted{
				{
					Namespace: app.Namespace,
					Name:      app.Name + "-ing",
					Resource:  "ingresses",
				},
			},
		},
	}

	f, err := os.OpenFile("../config/crd/ingresses.yaml", os.O_RDONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	it := testutil.NewIntegrationTester(f, newIngressExtension)
	it.Run(t, tests...)
}
