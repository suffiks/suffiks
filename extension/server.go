package extension

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"reflect"

	"github.com/suffiks/suffiks/extension/protogen"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type server[T any] struct {
	protogen.UnimplementedExtensionServer

	ext  Extension[T]
	vext ValidatableExtension[T]
	dext DefaultableExtension[T]
}

var _ protogen.ExtensionServer = &server[any]{}

func (s *server[T]) Sync(req *protogen.SyncRequest, e protogen.Extension_SyncServer) error {
	fmt.Println("Run Sync")
	rw := &ResponseWriter{w: e}

	var obj T
	if len(req.GetSpec()) > 0 {
		if err := json.Unmarshal(req.GetSpec(), &obj); err != nil {
			return err
		}
	}

	err := s.ext.Sync(e.Context(), Owner{owner: req.GetOwner()}, obj, rw)
	if err != nil {
		log.Println("sync error:", err)
		return err
	}

	return nil
}

func (s *server[T]) Delete(req *protogen.SyncRequest, e protogen.Extension_DeleteServer) error {
	fmt.Println("Run Delete")
	var obj T
	if len(req.GetSpec()) > 0 {
		if err := json.Unmarshal(req.GetSpec(), &obj); err != nil {
			return err
		}
	}

	err := s.ext.Delete(e.Context(), Owner{owner: req.GetOwner()}, obj)
	if err != nil {
		log.Println("delete error:", err)
		return err
	}
	return nil
}

func (s *server[T]) Default(ctx context.Context, req *protogen.SyncRequest) (*protogen.DefaultResponse, error) {
	fmt.Println("Run Default")
	if s.dext == nil {
		return nil, nil
	}

	var obj T
	if len(req.GetSpec()) > 0 {
		if err := json.Unmarshal(req.GetSpec(), &obj); err != nil {
			return nil, err
		}
	}

	def, err := s.dext.Default(ctx, Owner{owner: req.GetOwner()}, obj)
	if err != nil {
		log.Println("defaulting error:", err)
		return nil, err
	}

	spec, err := json.Marshal(def)
	if err != nil {
		log.Println("defaulting error:", err)
		return nil, err
	}

	resp := &protogen.DefaultResponse{
		Spec: spec,
	}

	return resp, nil
}

func (s *server[T]) Validate(ctx context.Context, req *protogen.ValidationRequest) (*protogen.ValidationResponse, error) {
	fmt.Println("Run Validate")
	if s.vext == nil {
		return &protogen.ValidationResponse{}, nil
	}

	newObject := instance[T]()
	oldObject := instance[T]()

	sync := req.GetSync()
	if sync != nil && len(sync.GetSpec()) > 0 {
		fmt.Println(string(sync.GetSpec()))
		if err := json.Unmarshal(sync.GetSpec(), newObject); err != nil {
			return nil, fmt.Errorf("error unmarshaling newObject: %w", err)
		}
	}

	old := req.GetOld()
	if old != nil && len(old.GetSpec()) > 0 {
		if err := json.Unmarshal(old.GetSpec(), oldObject); err != nil {
			return nil, fmt.Errorf("error unmarshaling oldObject: %w", err)
		}
	}

	valErrs, err := s.vext.Validate(ctx, ValidationType(req.Type), Owner{owner: sync.GetOwner()}, newObject, oldObject)
	if err != nil {
		log.Println("validation error:", err)
		return nil, err
	}

	resp := &protogen.ValidationResponse{}
	for _, valErr := range valErrs {
		resp.Errors = append(resp.Errors, &protogen.ValidationError{
			Path:   valErr.Path,
			Value:  valErr.Value,
			Detail: valErr.Detail,
		})
	}

	return resp, nil
}

func Serve[T any](ctx context.Context, address string, ext Extension[T]) error {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	s := grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
		grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()),
	)

	protogen.RegisterExtensionServer(s, NewServer(ext))

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return s.Serve(lis) })
	g.Go(func() error {
		<-ctx.Done()
		s.GracefulStop()
		return nil
	})

	return g.Wait()
}

func NewServer[T any](ext Extension[T]) protogen.ExtensionServer {
	vext, _ := ext.(ValidatableExtension[T])
	dext, _ := ext.(DefaultableExtension[T])
	return &server[T]{
		ext:  ext,
		vext: vext,
		dext: dext,
	}
}

func instance[T any]() T {
	var obj T
	return reflect.New(reflect.TypeOf(obj).Elem()).Interface().(T)
}
