package waruntime_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/suffiks/suffiks/extension/protogen"
	"github.com/suffiks/suffiks/internal/waruntime"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
)

func TestRun(t *testing.T) {
	ctx := context.Background()
	r := waruntime.New(ctx)
	defer r.Close(ctx)

	b, err := os.ReadFile("./testdata/as/build/debug.wasm")
	if err != nil {
		t.Fatal(err)
	}

	if err := r.Load(ctx, "test", "0.1.1", b); err != nil {
		t.Fatal(err)
	}

	runner, err := r.NewRunner(ctx, "test", nil)
	if err != nil {
		t.Fatal(err)
	}

	syncReq := &protogen.SyncRequest{
		Owner: &protogen.Owner{
			Kind:       "Application",
			Name:       "my-app",
			Namespace:  "some-namespace",
			ApiVersion: "suffiks.io/v1",
			Uid:        "some-uid",
			Labels: map[string]string{
				"app": "my-app",
			},
			Annotations: map[string]string{
				"app": "my-app",
			},
			RevisionID: "some-revision-id",
		},
		Spec: []byte(`{"ingresses":[{"host":"suffiks"}, {"host":"suffiks.com", "paths":["test"]}]}`),
	}

	errs, err := runner.Validate(ctx, &protogen.ValidationRequest{
		Type: protogen.ValidationType_CREATE,
		Sync: syncReq,
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := []*protogen.ValidationError{
		{
			Path:   "ingresses[0].host",
			Detail: "must contain a dot",
			Value:  "suffiks",
		},
		{
			Path:   "ingresses[1].paths[0]",
			Detail: "must start with a slash",
			Value:  "test",
		},
	}

	opts := []cmp.Option{
		cmpopts.IgnoreUnexported(protogen.ValidationError{}),
	}
	if len(errs) != len(expected) || !cmp.Equal(errs, expected, opts...) {
		t.Errorf("diff: -got +want\n%s", cmp.Diff(errs, expected, opts...))
	}

	// DEFAULTING
	runner, err = r.NewRunner(ctx, "test", nil)
	if err != nil {
		t.Fatal(err)
	}

	dr, err := runner.Defaulting(ctx, syncReq)
	if err != nil {
		t.Fatal(err)
	}

	expectedSpec := map[string]any{
		"ingresses": []any{
			map[string]any{
				"host":  "suffiks",
				"paths": []any{"//"},
			},
			map[string]any{
				"host":  "suffiks.com",
				"paths": []any{"test"},
			},
		},
	}

	got := map[string]any{}
	if err := json.Unmarshal(dr.Spec, &got); err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(got, expectedSpec) {
		t.Errorf("diff: -got +want\n%s", cmp.Diff(got, expectedSpec))
	}

	// SYNC
	t.Log("Sync")
	runner.Close(ctx)

	client := fake.NewSimpleDynamicClient(runtime.NewScheme())
	runner, err = r.NewRunner(ctx, "test", client)
	if err != nil {
		t.Fatal(err)
	}

	res, err := runner.Sync(ctx, syncReq)
	if err != nil {
		t.Fatal(err)
	}

	var gotResponses []*protogen.Response
	for {
		t.Log("waiting for recv")
		obj, err := res.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			t.Fatal(err)
		}

		gotResponses = append(gotResponses, obj)
	}

	expectedResponses := []*protogen.Response{
		{
			OFResponse: &protogen.Response_Label{
				Label: &protogen.KeyValue{
					Name:  "is-wasm-controlled",
					Value: "true",
				},
			},
		},
	}

	opts = []cmp.Option{
		cmpopts.IgnoreUnexported(protogen.Response{}, protogen.KeyValue{}, protogen.Response_Label{}),
	}
	if !cmp.Equal(gotResponses, expectedResponses, opts...) {
		t.Errorf("diff: -got +want\n%s", cmp.Diff(gotResponses, expectedResponses, opts...))
	}

	actions := client.Actions()
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}

	// SYNC AGAIN
	t.Log("Sync 2")
	runner.Close(ctx)

	runner, err = r.NewRunner(ctx, "test", client)
	if err != nil {
		t.Fatal(err)
	}

	res, err = runner.Sync(ctx, syncReq)
	if err != nil {
		t.Fatal(err)
	}

	gotResponses = nil
	for {
		t.Log("waiting for recv")
		obj, err := res.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			t.Fatal(err)
		}

		gotResponses = append(gotResponses, obj)
	}

	expectedResponses = []*protogen.Response{
		{
			OFResponse: &protogen.Response_Label{
				Label: &protogen.KeyValue{
					Name:  "is-wasm-controlled",
					Value: "true",
				},
			},
		},
	}

	opts = []cmp.Option{
		cmpopts.IgnoreUnexported(protogen.Response{}, protogen.KeyValue{}, protogen.Response_Label{}),
	}
	if !cmp.Equal(gotResponses, expectedResponses, opts...) {
		t.Errorf("diff: -got +want\n%s", cmp.Diff(gotResponses, expectedResponses, opts...))
	}

	actions = client.Actions()
	if len(actions) != 4 {
		t.Fatalf("expected 4 action, got %d", len(actions))
	}
}
