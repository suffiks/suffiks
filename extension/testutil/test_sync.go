package testutil

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/suffiks/suffiks/base"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
)

type SyncTest struct {
	Name      string
	Existing  []runtime.Object
	Object    base.Object
	Expected  []runtime.Object
	ErrCheck  func(t *testing.T, err error)
	Changeset *base.Changeset
}

func (s SyncTest) name() string               { return s.Name }
func (s SyncTest) existing() []runtime.Object { return s.Existing }
func (s SyncTest) runTest(t *testing.T, ctrl *base.ExtensionController, client *fake.Clientset) {
	t.Helper()

	cs, err := ctrl.Sync(context.Background(), fixObject(t, s.Object))
	if err != nil {
		if s.ErrCheck == nil {
			t.Fatal(err)
		} else {
			s.ErrCheck(t, err)
		}
	}

	if s.Changeset != nil && !cmp.Equal(s.Changeset, cs.Changeset, cmpopts.IgnoreUnexported(base.Changeset{})) {
		t.Error(cmp.Diff(s.Changeset, cs.Changeset, cmpopts.IgnoreUnexported(base.Changeset{})))
	}

	for _, exp := range s.Expected {
		ns := getNamespace(exp)
		name := getName(exp)

		resourceMap := map[string]schema.GroupVersionResource{}

		type getObject interface {
			GetObject() runtime.Object
		}
		for _, a := range client.Actions() {
			action, ok := a.(getObject)
			if !ok {
				continue
			}
			resourceMap[fmt.Sprintf("%T", action.GetObject())] = a.GetResource()
		}

		found, err := client.Tracker().Get(resourceMap[fmt.Sprintf("%T", exp)], ns, name)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(exp, found) {
			t.Error(cmp.Diff(exp, found))
		}
	}
}
