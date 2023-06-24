package waruntime

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"unicode"

	"github.com/suffiks/suffiks/extension/protogen"
	"github.com/suffiks/suffiks/internal/tracing"
	suffiksv1 "github.com/suffiks/suffiks/pkg/api/suffiks/v1"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

type Responder interface {
	Recv() (*protogen.Response, error)
}

type Runner struct {
	name       string
	version    string
	runtime    wazero.Runtime
	controller *Controller
	module     wazero.CompiledModule

	validationRequest *protogen.ValidationRequest
	syncRequest       *protogen.SyncRequest

	client             dynamic.Interface
	clientPermissions  map[string]struct{}
	configMapReference *suffiksv1.ConfigMapReference

	msgs             chan *protogen.Response
	lock             sync.Mutex
	validationErrors []*protogen.ValidationError
}

func (r *Runner) Close(ctx context.Context) error {
	// r.module.Close(ctx)
	// return r.runtime.Close(ctx)
	return nil
}

func (r *Runner) spanAttributes(span trace.Span) {
	span.SetAttributes(
		attribute.String("name", r.name),
		attribute.String("version", r.version),
	)
}

// env returns a map of functions that are exposed to the WASI module.
func (r *Runner) env() map[string]any {
	return map[string]any{
		"AddEnv":           r.addEnv,
		"AddEnvFrom":       r.addEnvFrom,
		"AddLabel":         r.addLabel,
		"AddAnnotation":    r.addAnnotation,
		"AddInitContainer": r.addInitContainer,
		"AddSidecar":       r.addSidecar,
		"MergePatch":       r.mergePatch,
		"ValidationError":  r.validationError,
		"GetOwner":         r.getOwner,
		"GetSpec":          r.getSpec,
		"GetOld":           r.getOld,
		"CreateResource":   r.createResource,
		"UpdateResource":   r.updateResource,
		"DeleteResource":   r.deleteResource,
		"GetResource":      r.getResource,
	}
}

func (r *Runner) instance(ctx context.Context) (api.Module, error) {
	ctx, span := tracing.Start(ctx, "WASI.Instance")
	defer span.End()
	r.spanAttributes(span)

	mod := r.runtime.NewHostModuleBuilder("suffiks")

	for name, fn := range r.env() {
		mod = mod.NewFunctionBuilder().WithFunc(fn).Export(name)
	}

	if _, err := mod.Instantiate(ctx); err != nil {
		return nil, fmt.Errorf("instantiate: %w", err)
	}

	cfg := wazero.NewModuleConfig().
		WithStdout(os.Stdout).
		WithStderr(os.Stderr)

	if r.configMapReference != nil {
		cmu, err := r.client.Resource(schema.GroupVersionResource{
			Group:    "",
			Version:  "v1",
			Resource: "configmaps",
		}).Namespace(r.configMapReference.Namespace).Get(ctx, r.configMapReference.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("get configmap: %w", err)
		}

		var cm corev1.ConfigMap
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(cmu.Object, &cm); err != nil {
			return nil, fmt.Errorf("convert configmap: %w", err)
		}

		for k, v := range cm.Data {
			if isUpper(k) {
				cfg = cfg.WithEnv(k, v)
			}
		}
	}

	return r.runtime.InstantiateModule(ctx, r.module, cfg)
}

func (r *Runner) Validate(ctx context.Context, req *protogen.ValidationRequest) ([]*protogen.ValidationError, error) {
	ctx, span := tracing.Start(ctx, "WASI.Validate")
	defer span.End()
	r.spanAttributes(span)

	mod, err := r.instance(ctx)
	if err != nil {
		return nil, err
	}
	defer mod.Close(ctx)

	r.validationRequest = req
	typ := uint64(req.Type)
	_, err = mod.ExportedFunction("Validate").Call(ctx, typ)

	r.lock.Lock()
	defer r.lock.Unlock()
	return r.validationErrors[:], err
}

func (r *Runner) Defaulting(ctx context.Context, req *protogen.SyncRequest) (*protogen.DefaultResponse, error) {
	ctx, span := tracing.Start(ctx, "WASI.Defaulting")
	defer span.End()
	r.spanAttributes(span)

	mod, err := r.instance(ctx)
	if err != nil {
		return nil, err
	}
	defer mod.Close(ctx)

	r.syncRequest = req
	ret, err := mod.ExportedFunction("Defaulting").Call(ctx)
	if err != nil {
		return nil, err
	}

	ptrAndSize := uint64(ret[0])
	if ptrAndSize == 0 {
		return &protogen.DefaultResponse{}, nil
	}

	ptr := ptrAndSize >> 32

	b, ok := mod.Memory().Read(uint32(ptr), uint32(ptrAndSize))
	if !ok {
		return nil, fmt.Errorf("failed to read memory at %d with size %d", ptr, uint32(ptrAndSize))
	}

	return &protogen.DefaultResponse{Spec: b}, nil
}

type response struct {
	chn    chan *protogen.Response
	errors chan error
}

func (r *response) Recv() (*protogen.Response, error) {
	for {
		select {
		case msg, ok := <-r.chn:
			if !ok {
				return nil, io.EOF
			}
			return msg, nil
		case err, ok := <-r.errors:
			if !ok {
				return nil, io.EOF
			}
			return nil, err
		}
	}
}

func (r *Runner) Sync(ctx context.Context, req *protogen.SyncRequest) (Responder, error) {
	ctx, span := tracing.Start(ctx, "WASI.Sync")
	defer span.End()
	r.spanAttributes(span)

	res := &response{
		chn:    make(chan *protogen.Response, 1),
		errors: make(chan error, 1),
	}
	r.msgs = res.chn

	r.syncRequest = req

	mod, err := r.instance(ctx)
	if err != nil {
		return nil, err
	}

	go func() {
		defer mod.Close(ctx)

		_, err = mod.ExportedFunction("Sync").Call(ctx)
		if err != nil {
			res.errors <- err
		}

		close(r.msgs)
		close(res.errors)
	}()

	return res, nil
}

func (r *Runner) Delete(ctx context.Context, req *protogen.SyncRequest) error {
	ctx, span := tracing.Start(ctx, "WASI.Delete")
	defer span.End()
	r.spanAttributes(span)

	mod, err := r.instance(ctx)
	if err != nil {
		return err
	}
	defer mod.Close(ctx)

	r.syncRequest = req
	_, err = mod.ExportedFunction("Delete").Call(ctx)
	return err
}

func (r *Runner) addEnv(ctx context.Context, m api.Module, ptr, size uint32) {
	span := tracing.Get(ctx)
	span.AddEvent("addEnv")
	r.msgs <- &protogen.Response{
		OFResponse: &protogen.Response_Env{
			Env: unmarshalProto(m, &protogen.KeyValue{}, ptr, size),
		},
	}
}

func (r *Runner) addEnvFrom(ctx context.Context, m api.Module, ptr, size uint32) {
	span := tracing.Get(ctx)
	span.AddEvent("addEnvFrom")
	r.msgs <- &protogen.Response{
		OFResponse: &protogen.Response_EnvFrom{
			EnvFrom: unmarshalProto(m, &protogen.EnvFrom{}, ptr, size),
		},
	}
}

func (r *Runner) addLabel(ctx context.Context, m api.Module, ptr, size uint32) {
	span := tracing.Get(ctx)
	span.AddEvent("addLabel")
	r.msgs <- &protogen.Response{
		OFResponse: &protogen.Response_Label{
			Label: unmarshalProto(m, &protogen.KeyValue{}, ptr, size),
		},
	}
}

func (r *Runner) addAnnotation(ctx context.Context, m api.Module, ptr, size uint32) {
	span := tracing.Get(ctx)
	span.AddEvent("addAnnotation")
	r.msgs <- &protogen.Response{
		OFResponse: &protogen.Response_Annotation{
			Annotation: unmarshalProto(m, &protogen.KeyValue{}, ptr, size),
		},
	}
}

func (r *Runner) addInitContainer(ctx context.Context, m api.Module, ptr, size uint32) {
	span := tracing.Get(ctx)
	span.AddEvent("addInitContainer")
	r.msgs <- &protogen.Response{
		OFResponse: &protogen.Response_InitContainer{
			InitContainer: unmarshalProto(m, &protogen.Container{}, ptr, size),
		},
	}
}

func (r *Runner) addSidecar(ctx context.Context, m api.Module, ptr, size uint32) {
	span := tracing.Get(ctx)
	span.AddEvent("addSidecar")
	r.msgs <- &protogen.Response{
		OFResponse: &protogen.Response_Container{
			Container: unmarshalProto(m, &protogen.Container{}, ptr, size),
		},
	}
}

func (r *Runner) mergePatch(ctx context.Context, m api.Module, ptr, size uint32) {
	span := tracing.Get(ctx)
	span.AddEvent("addMergePatch")
	b, ok := m.Memory().Read(ptr, size)
	if !ok {
		panic("failed to read memory")
	}

	r.msgs <- &protogen.Response{
		OFResponse: &protogen.Response_MergePatch{
			MergePatch: b,
		},
	}
}

func (r *Runner) getOwner(ctx context.Context, m api.Module) uint64 {
	span := tracing.Get(ctx)
	span.AddEvent("getOwner")

	var owner *protogen.Owner

	if r.syncRequest != nil {
		owner = r.syncRequest.Owner
	} else if r.validationRequest != nil {
		owner = r.validationRequest.Sync.Owner
	} else {
		panic("getOwner is only valid for sync or validation requests")
	}

	return marshalProto(ctx, m, owner)
}

func (r *Runner) getSpec(ctx context.Context, m api.Module) uint64 {
	span := tracing.Get(ctx)
	span.AddEvent("getSpec")

	var b []byte
	if r.syncRequest != nil {
		b = r.syncRequest.Spec
	} else if r.validationRequest != nil && r.validationRequest.Sync != nil {
		b = r.validationRequest.Sync.Spec
	} else {
		panic("getSpec is not valid in this context")
	}

	return writeByteSlice(ctx, m, b)
}

func (r *Runner) getOld(ctx context.Context, m api.Module) uint64 {
	span := tracing.Get(ctx)
	span.AddEvent("getOld")

	if r.validationRequest == nil {
		panic("getOld is only valid for validation requests")
	}

	return writeByteSlice(ctx, m, r.validationRequest.Old.Spec)
}

func (r *Runner) validationError(ctx context.Context, m api.Module, ptr, size uint32) {
	span := tracing.Get(ctx)
	span.AddEvent("validationError")

	r.lock.Lock()
	defer r.lock.Unlock()

	r.validationErrors = append(
		r.validationErrors,
		unmarshalProto(m, &protogen.ValidationError{}, ptr, size),
	)
}

// getResource returns a resource from the Kubernetes API server.
func (r *Runner) getResource(ctx context.Context, m api.Module, gvrPtr, gvrSize, namePtr, nameSize uint32) uint64 {
	ctx, span := tracing.Start(ctx, "WASI.GetResource")
	defer span.End()
	r.spanAttributes(span)

	gvr := unmarshalProto(m, &protogen.GroupVersionResource{}, gvrPtr, gvrSize)
	if err := r.isAllowed(ctx, gvr, "get"); err != nil {
		log.Println(err)
		return uint64(toClientError(err))
	}

	nameb, ok := m.Memory().Read(namePtr, nameSize)
	if !ok {
		panic("failed to read memory")
	}

	span.SetAttributes(attribute.String("resource.name", string(nameb)), attribute.String("resource.namespace", r.syncRequest.Owner.Namespace))

	resource, err := r.client.Resource(schema.GroupVersionResource{
		Group:    gvr.GetGroup(),
		Version:  gvr.GetVersion(),
		Resource: gvr.GetResource(),
	}).Namespace(r.syncRequest.Owner.Namespace).Get(ctx, string(nameb), metav1.GetOptions{})
	if err != nil {
		log.Println(err)
		return uint64(toClientError(err))
	}

	b, err := resource.MarshalJSON()
	if err != nil {
		log.Println(err)
		return uint64(toClientError(err))
	}

	return writeByteSlice(ctx, m, b)
}

func (r *Runner) deleteResource(ctx context.Context, m api.Module, gvrPtr, gvrSize, namePtr, nameSize uint32) uint64 {
	ctx, span := tracing.Start(ctx, "WASI.DeleteResource")
	defer span.End()
	r.spanAttributes(span)

	gvr := unmarshalProto(m, &protogen.GroupVersionResource{}, gvrPtr, gvrSize)
	if err := r.isAllowed(ctx, gvr, "delete"); err != nil {
		log.Println(err)
		return uint64(toClientError(err))
	}

	nameb, ok := m.Memory().Read(namePtr, nameSize)
	if !ok {
		panic("failed to read memory")
	}

	span.SetAttributes(attribute.String("resource.name", string(nameb)), attribute.String("resource.namespace", r.syncRequest.Owner.Namespace))

	err := r.client.Resource(schema.GroupVersionResource{
		Group:    gvr.GetGroup(),
		Version:  gvr.GetVersion(),
		Resource: gvr.GetResource(),
	}).Namespace(r.syncRequest.Owner.Namespace).Delete(ctx, string(nameb), metav1.DeleteOptions{})
	if err != nil {
		log.Println(err)
		return uint64(toClientError(err))
	}
	return 0
}

func (r *Runner) createResource(ctx context.Context, m api.Module, gvrPtr, gvrSize, specPtr, specSize uint32) uint64 {
	ctx, span := tracing.Start(ctx, "WASI.CreateResource")
	defer span.End()
	r.spanAttributes(span)

	gvr := unmarshalProto(m, &protogen.GroupVersionResource{}, gvrPtr, gvrSize)
	if err := r.isAllowed(ctx, gvr, "create"); err != nil {
		log.Println(err)
		return uint64(toClientError(err))
	}

	b, ok := m.Memory().Read(specPtr, specSize)
	if !ok {
		panic("failed to read memory")
	}

	resource := &unstructured.Unstructured{}
	if err := resource.UnmarshalJSON(b); err != nil {
		panic("failed to unmarshal resource: " + err.Error())
	}

	span.SetAttributes(attribute.String("resource.name", resource.GetName()), attribute.String("resource.namespace", r.syncRequest.Owner.Namespace))
	// TODO: Is there some way to create a dynamic lister for any resouce requested?
	n, err := r.client.Resource(schema.GroupVersionResource{
		Group:    gvr.GetGroup(),
		Version:  gvr.GetVersion(),
		Resource: gvr.GetResource(),
	}).Namespace(r.syncRequest.Owner.Namespace).Create(ctx, resource, metav1.CreateOptions{})
	if err != nil {
		log.Println(err)
		return uint64(toClientError(err))
	}

	b, err = n.MarshalJSON()
	if err != nil {
		panic("failed to marshal resource: " + err.Error())
	}

	return writeByteSlice(ctx, m, b)
}

func (r *Runner) updateResource(ctx context.Context, m api.Module, gvrPtr, gvrSize, specPtr, specSize uint32) uint64 {
	ctx, span := tracing.Start(ctx, "WASI.UpdateResource")
	defer span.End()
	r.spanAttributes(span)

	gvr := unmarshalProto(m, &protogen.GroupVersionResource{}, gvrPtr, gvrSize)
	if err := r.isAllowed(ctx, gvr, "update"); err != nil {
		log.Println(err)
		return uint64(toClientError(err))
	}

	b, ok := m.Memory().Read(specPtr, specSize)
	if !ok {
		panic("failed to read memory")
	}

	resource := &unstructured.Unstructured{}
	if err := resource.UnmarshalJSON(b); err != nil {
		panic("failed to unmarshal resource: " + err.Error())
	}

	span.SetAttributes(attribute.String("resource.name", resource.GetName()), attribute.String("resource.namespace", r.syncRequest.Owner.Namespace))
	// TODO: Is there some way to create a dynamic lister for any resouce requested?
	n, err := r.client.Resource(schema.GroupVersionResource{
		Group:    gvr.GetGroup(),
		Version:  gvr.GetVersion(),
		Resource: gvr.GetResource(),
	}).Namespace(r.syncRequest.Owner.Namespace).Update(ctx, resource, metav1.UpdateOptions{})
	if err != nil {
		log.Println(err)
		return uint64(toClientError(err))
	}

	b, err = n.MarshalJSON()
	if err != nil {
		panic("failed to marshal resource: " + err.Error())
	}

	return writeByteSlice(ctx, m, b)
}

func (r *Runner) isAllowed(ctx context.Context, gvr *protogen.GroupVersionResource, method string) error {
	span := tracing.Get(ctx)
	span.SetAttributes(
		attribute.String("resource.group", gvr.GetGroup()),
		attribute.String("resource.version", gvr.GetVersion()),
		attribute.String("resource.resource", gvr.GetResource()),
		attribute.String("resource.method", method),
	)

	if r.clientPermissions != nil {
		_, ok := r.clientPermissions[gvr.GetGroup()+"/"+gvr.GetVersion()+"/"+gvr.GetResource()+"."+method]
		if ok {
			return nil
		}
	}

	err := fmt.Errorf("extension is not configured to access this resource with method %q", method)
	span.RecordError(err)
	return apierrors.NewForbidden(schema.GroupResource{Group: gvr.GetGroup(), Resource: gvr.GetResource()}, "", err)
}

func unmarshalProto[T protoreflect.ProtoMessage](m api.Module, v T, ptr, size uint32) T {
	b, ok := m.Memory().Read(ptr, size)
	if !ok {
		panic("failed to read memory")
	}

	if err := proto.Unmarshal(b, v); err != nil {
		panic("unable to unmarshal: " + err.Error())
	}
	return v
}

func marshalProto(ctx context.Context, m api.Module, v protoreflect.ProtoMessage) uint64 {
	b, err := proto.Marshal(v)
	if err != nil {
		panic("marshalProto: " + err.Error())
	}
	return writeByteSlice(ctx, m, b)
}

func writeByteSlice(ctx context.Context, m api.Module, b []byte) uint64 {
	res, err := m.ExportedFunction("malloc").Call(ctx, uint64(len(b)))
	if err != nil {
		panic("marshalProto: " + err.Error())
	}

	ptr := res[0]
	fmt.Println("######### ptr: ", uint32(ptr), " len(b): ", uint32(len(b)))
	fmt.Println("######### Verify: ", (uint32(ptr)<<16)>>16)
	if ok := m.Memory().Write(uint32(ptr), b); !ok {
		panic("marshalProto: unable to write to memory")
	}

	return uint64(ptr)<<32 | uint64(len(b))
}

// func ptrSizeToString(mod api.Module, ptrSize uint32) (string, error) {
// 	size := ptrSize & 0xFFFF
// 	ptr := ptrSize >> 16

// 	b, ok := mod.Memory().Read(uint32(ptr), uint32(size))
// 	if !ok {
// 		return "", fmt.Errorf("failed to read memory at %d with size %d", ptr, size)
// 	}

// 	return string(b), nil
// }

func isUpper(s string) bool {
	for _, r := range s {
		if !unicode.IsUpper(r) && unicode.IsLetter(r) {
			return false
		}
	}
	return true
}
