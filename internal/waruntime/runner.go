package waruntime

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	//lint:ignore SA1019 Keep using module since it's still being used by client-go (their proto types)
	golangproto "github.com/golang/protobuf/proto"

	"github.com/suffiks/suffiks/extension/protogen"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

type Responder interface {
	Recv() (*protogen.Response, error)
}

type Runner struct {
	runtime    wazero.Runtime
	controller *Controller
	module     wazero.CompiledModule

	validationRequest *protogen.ValidationRequest
	syncRequest       *protogen.SyncRequest

	client dynamic.Interface

	msgs             chan *protogen.Response
	lock             sync.Mutex
	validationErrors []*protogen.ValidationError
}

func (r *Runner) Close(ctx context.Context) error {
	// r.module.Close(ctx)
	// return r.runtime.Close(ctx)
	return nil
}

func (r *Runner) instance(ctx context.Context) (api.Module, error) {
	mod := r.runtime.NewHostModuleBuilder("suffiks")

	funcs := map[string]any{
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
		// "DeleteResource": r.deleteResource,
		"GetResource": r.getResource,
	}
	for name, fn := range funcs {
		mod = mod.NewFunctionBuilder().WithFunc(fn).Export(name)
	}

	if _, err := mod.Instantiate(ctx); err != nil {
		return nil, fmt.Errorf("instantiate: %w", err)
	}

	cfg := wazero.NewModuleConfig().
		WithStdout(os.Stdout).
		WithStderr(os.Stderr)

	return r.runtime.InstantiateModule(ctx, r.module, cfg)
}

func (r *Runner) Validate(ctx context.Context, req *protogen.ValidationRequest) ([]*protogen.ValidationError, error) {
	mod, err := r.instance(ctx)
	if err != nil {
		return nil, err
	}
	defer mod.Close(ctx)

	r.validationRequest = req
	_, err = mod.ExportedFunction("Validate").Call(ctx, uint64(req.Type))

	r.lock.Lock()
	defer r.lock.Unlock()
	return r.validationErrors[:], err
}

func (r *Runner) Defaulting(ctx context.Context, req *protogen.SyncRequest) (*protogen.DefaultResponse, error) {
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

	ptrAndSize := uint32(ret[0])
	if ptrAndSize == 0 {
		return &protogen.DefaultResponse{}, nil
	}

	size := ptrAndSize & 0xFFFF
	ptr := ptrAndSize >> 16

	b, ok := mod.Memory().Read(uint32(ptr), uint32(size))
	if !ok {
		return nil, fmt.Errorf("failed to read memory at %d with size %d", ptr, size)
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
	mod, err := r.instance(ctx)
	if err != nil {
		return err
	}
	defer mod.Close(ctx)

	r.syncRequest = req
	_, err = mod.ExportedFunction("Delete").Call(ctx)
	return err
}

func (r *Runner) addEnv(_ context.Context, m api.Module, ptr, size uint32) {
	r.msgs <- &protogen.Response{
		OFResponse: &protogen.Response_Env{
			Env: unmarshalProto(m, &protogen.KeyValue{}, ptr, size),
		},
	}
}

func (r *Runner) addEnvFrom(_ context.Context, m api.Module, ptr, size uint32) {
	r.msgs <- &protogen.Response{
		OFResponse: &protogen.Response_EnvFrom{
			EnvFrom: unmarshalProto(m, &protogen.EnvFrom{}, ptr, size),
		},
	}
}

func (r *Runner) addLabel(_ context.Context, m api.Module, ptr, size uint32) {
	r.msgs <- &protogen.Response{
		OFResponse: &protogen.Response_Label{
			Label: unmarshalProto(m, &protogen.KeyValue{}, ptr, size),
		},
	}
}

func (r *Runner) addAnnotation(_ context.Context, m api.Module, ptr, size uint32) {
	r.msgs <- &protogen.Response{
		OFResponse: &protogen.Response_Annotation{
			Annotation: unmarshalProto(m, &protogen.KeyValue{}, ptr, size),
		},
	}
}

func (r *Runner) addInitContainer(_ context.Context, m api.Module, ptr, size uint32) {
	r.msgs <- &protogen.Response{
		OFResponse: &protogen.Response_InitContainer{
			InitContainer: unmarshalClientGoProto(m, &v1.Container{}, ptr, size),
		},
	}
}

func (r *Runner) addSidecar(_ context.Context, m api.Module, ptr, size uint32) {
	r.msgs <- &protogen.Response{
		OFResponse: &protogen.Response_Container{
			Container: unmarshalClientGoProto(m, &v1.Container{}, ptr, size),
		},
	}
}

func (r *Runner) mergePatch(_ context.Context, m api.Module, ptr, size uint32) {
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

func (r *Runner) getOwner(ctx context.Context, m api.Module) uint32 {
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

func (r *Runner) getSpec(ctx context.Context, m api.Module) uint32 {
	var b []byte
	if r.syncRequest != nil {
		b = r.syncRequest.Spec
	} else if r.validationRequest != nil {
		b = r.validationRequest.Sync.Spec
	} else {
		panic("getSpec is only valid in this context")
	}

	return writeByteSlice(ctx, m, b)
}

func (r *Runner) getOld(ctx context.Context, m api.Module) uint32 {
	if r.validationRequest == nil {
		panic("getOld is only valid for validation requests")
	}

	return writeByteSlice(ctx, m, r.validationRequest.Old.Spec)
}

func (r *Runner) validationError(_ context.Context, m api.Module, ptr, size uint32) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.validationErrors = append(
		r.validationErrors,
		unmarshalProto(m, &protogen.ValidationError{}, ptr, size),
	)
}

func (r *Runner) getResource(ctx context.Context, m api.Module, gvrPtr, gvrSize, namePtr, nameSize uint32) uint32 {
	nameb, ok := m.Memory().Read(namePtr, nameSize)
	if !ok {
		panic("failed to read memory")
	}

	gvr := unmarshalClientGoProto(m, &metav1.GroupVersionResource{}, gvrPtr, gvrSize)
	resource, err := r.client.Resource(schema.GroupVersionResource{
		Group:    gvr.Group,
		Version:  gvr.Version,
		Resource: gvr.Resource,
	}).Namespace(r.syncRequest.Owner.Namespace).Get(ctx, string(nameb), metav1.GetOptions{})
	if err != nil {
		panic("failed to get resource: " + err.Error())
	}

	b, err := resource.MarshalJSON()
	if err != nil {
		log.Println(err)
		return uint32(toClientError(err))
	}

	return writeByteSlice(ctx, m, b)
}

func (r *Runner) createResource(ctx context.Context, m api.Module, gvrPtr, gvrSize, specPtr, specSize uint32) uint32 {
	gvr := unmarshalClientGoProto(m, &metav1.GroupVersionResource{}, gvrPtr, gvrSize)
	b, ok := m.Memory().Read(specPtr, specSize)
	if !ok {
		panic("failed to read memory")
	}

	resource := &unstructured.Unstructured{}
	if err := resource.UnmarshalJSON(b); err != nil {
		panic("failed to unmarshal resource: " + err.Error())
	}

	// TODO: Is there some way to create a dynamic lister for any resouce requested?
	n, err := r.client.Resource(schema.GroupVersionResource{
		Group:    gvr.Group,
		Version:  gvr.Version,
		Resource: gvr.Resource,
	}).Namespace(r.syncRequest.Owner.Namespace).Create(ctx, resource, metav1.CreateOptions{})
	if err != nil {
		log.Println(err)
		return uint32(toClientError(err))
	}

	b, err = n.MarshalJSON()
	if err != nil {
		panic("failed to marshal resource: " + err.Error())
	}

	return writeByteSlice(ctx, m, b)
}

func (r *Runner) updateResource(ctx context.Context, m api.Module, gvrPtr, gvrSize, specPtr, specSize uint32) uint32 {
	gvr := unmarshalClientGoProto(m, &metav1.GroupVersionResource{}, gvrPtr, gvrSize)
	b, ok := m.Memory().Read(specPtr, specSize)
	if !ok {
		panic("failed to read memory")
	}

	resource := &unstructured.Unstructured{}
	if err := resource.UnmarshalJSON(b); err != nil {
		panic("failed to unmarshal resource: " + err.Error())
	}

	// TODO: Is there some way to create a dynamic lister for any resouce requested?
	n, err := r.client.Resource(schema.GroupVersionResource{
		Group:    gvr.Group,
		Version:  gvr.Version,
		Resource: gvr.Resource,
	}).Namespace(r.syncRequest.Owner.Namespace).Update(ctx, resource, metav1.UpdateOptions{})
	if err != nil {
		log.Println(err)
		return uint32(toClientError(err))
	}

	b, err = n.MarshalJSON()
	if err != nil {
		panic("failed to marshal resource: " + err.Error())
	}

	return writeByteSlice(ctx, m, b)
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

func unmarshalClientGoProto[T golangproto.Message](m api.Module, v T, ptr, size uint32) T {
	b, ok := m.Memory().Read(ptr, size)
	if !ok {
		panic("failed to read memory")
	}

	if err := golangproto.Unmarshal(b, v); err != nil {
		panic("unable to unmarshal: " + err.Error())
	}
	return v
}

func marshalProto(ctx context.Context, m api.Module, v protoreflect.ProtoMessage) uint32 {
	b, err := proto.Marshal(v)
	if err != nil {
		panic("marshalProto: " + err.Error())
	}
	return writeByteSlice(ctx, m, b)
}

func writeByteSlice(ctx context.Context, m api.Module, b []byte) uint32 {
	res, err := m.ExportedFunction("Malloc").Call(ctx, uint64(len(b)))
	if err != nil {
		panic("marshalProto: " + err.Error())
	}

	ptr := res[0]
	if ok := m.Memory().Write(uint32(ptr), b); !ok {
		panic("marshalProto: unable to write to memory")
	}

	return uint32(ptr)<<16 | uint32(len(b))
}

func ptrSizeToString(mod api.Module, ptrSize uint32) (string, error) {
	size := ptrSize & 0xFFFF
	ptr := ptrSize >> 16

	b, ok := mod.Memory().Read(uint32(ptr), uint32(size))
	if !ok {
		return "", fmt.Errorf("failed to read memory at %d with size %d", ptr, size)
	}

	return string(b), nil
}
