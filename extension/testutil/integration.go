package testutil

import (
	"context"
	"io"
	"testing"

	"github.com/suffiks/suffiks/extension"
	controller "github.com/suffiks/suffiks/internal/controller"
	suffiksv1 "github.com/suffiks/suffiks/pkg/api/suffiks/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

type Action int

const (
	Sync Action = iota
	Delete
	Validate
	Default
)

type SetupExtension[Ext any] func(client *fake.Clientset) extension.Extension[Ext]

type IntegrationTester[Ext any] struct {
	setupExtension SetupExtension[Ext]
	spec           io.Reader
}

func NewIntegrationTester[Ext any](spec io.Reader, extSetup SetupExtension[Ext]) *IntegrationTester[Ext] {
	return &IntegrationTester[Ext]{
		setupExtension: extSetup,
		spec:           spec,
	}
}

func (i *IntegrationTester[Ext]) Run(t *testing.T, tests ...TestCase) {
	t.Helper()

	existing := []runtime.Object{}
	for _, test := range tests {
		existing = append(existing, test.existing()...)
	}

	client := fake.NewSimpleClientset(existing...)
	ext := i.setupExtension(client)
	tr, err := New(ext, i.spec)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if err := tr.Run(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tr.Stop() }()

	for _, test := range tests {
		t.Run(test.name(), func(t *testing.T) {
			i.runTest(t, client, tr, test)
		})
	}
}

func (i *IntegrationTester[Ext]) runTest(t *testing.T, client *fake.Clientset, tr *Suffiks[Ext], test TestCase) {
	t.Helper()

	test.runTest(t, tr.ctrl, client)
}

type TestCase interface {
	runTest(t *testing.T, ctrl *controller.ExtensionController, client *fake.Clientset)
	name() string
	existing() []runtime.Object
}

func getNamespace(obj runtime.Object) string {
	type v interface {
		GetNamespace() string
	}

	ns, ok := obj.(v)
	if !ok {
		return ""
	}
	return ns.GetNamespace()
}

func getName(obj runtime.Object) string {
	type v interface {
		GetName() string
	}

	ns, ok := obj.(v)
	if !ok {
		return ""
	}
	return ns.GetName()
}

func fixObject(t *testing.T, obj controller.Object) controller.Object {
	if obj.GetObjectKind().GroupVersionKind().Kind == "" {
		switch v := obj.(type) {
		case *suffiksv1.Application:
			v.Kind = "Application"
			v.APIVersion = suffiksv1.GroupVersion.Version
			return v
		case *suffiksv1.Work:
			v.Kind = "Work"
			v.APIVersion = suffiksv1.GroupVersion.Version
			return v
		default:
			t.Fatalf("Unknown object type %T. You can set the object kind to continue", obj)
		}
	}
	return obj
}
