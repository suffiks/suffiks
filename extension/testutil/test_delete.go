package testutil

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	controller "github.com/suffiks/suffiks/internal/controllers"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

type DeleteTest struct {
	Name string
	// Existing is the list of objects that should exist before the test is run.
	Existing []runtime.Object
	Object   controller.Object
	// Expected is the list of expected resources after the test case.
	ExpectedRemaining []runtime.Object
	ExpectedDeleted   []Deleted
	ErrCheck          func(t *testing.T, err error)
}

func (s DeleteTest) name() string               { return s.Name }
func (s DeleteTest) existing() []runtime.Object { return s.Existing }
func (s DeleteTest) runTest(t *testing.T, ctrl *controller.ExtensionController, client *fake.Clientset) {
	if err := ctrl.Delete(context.Background(), fixObject(t, s.Object)); err != nil {
		if s.ErrCheck == nil {
			t.Fatal(err)
		} else {
			s.ErrCheck(t, err)
		}
	}

	resourceMap := map[string]schema.GroupVersionResource{}

	type getObject interface {
		GetObject() runtime.Object
	}

	toDelete := map[string]struct{}{}
	for _, om := range s.ExpectedDeleted {
		toDelete[om.key()] = struct{}{}
	}

	for _, a := range client.Actions() {
		if da, ok := a.(clienttesting.DeleteAction); ok {
			for _, o := range s.ExpectedDeleted {
				if o.matches(da) {
					delete(toDelete, o.key())
				}
			}
		}

		action, ok := a.(getObject)
		if !ok {
			continue
		}

		resourceMap[fmt.Sprintf("%T", action.GetObject())] = a.GetResource()
	}

	for td := range toDelete {
		t.Error(td + " not deleted")
	}

	for _, exp := range s.ExpectedRemaining {
		ns := getNamespace(exp)
		name := getName(exp)

		found, err := client.Tracker().Get(resourceMap[fmt.Sprintf("%T", exp)], ns, name)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(exp, found) {
			t.Error(cmp.Diff(exp, found))
		}
	}
}

type Deleted struct {
	Namespace string
	Name      string
	Resource  string
}

func (d Deleted) key() string {
	return d.Resource + " " + d.Namespace + "/" + d.Name
}

func (d Deleted) matches(da clienttesting.DeleteAction) bool {
	if d.Namespace != da.GetNamespace() {
		return false
	}
	if d.Name != da.GetName() {
		return false
	}
	if d.Resource != da.GetResource().Resource {
		return false
	}
	return true
}
