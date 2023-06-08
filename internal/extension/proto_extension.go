package extension

import (
	"context"

	suffiksv1 "github.com/suffiks/suffiks/api/suffiks/v1"
	"github.com/suffiks/suffiks/extension/protogen"
	"google.golang.org/grpc"
)

type Extension interface {
	Name() string
	Spec() suffiksv1.ExtensionSpec
	RootKeys() []string
	Close() error

	Default(ctx context.Context, in *protogen.SyncRequest) (*protogen.DefaultResponse, error)
	Validate(ctx context.Context, in *protogen.ValidationRequest) (*protogen.ValidationResponse, error)
	Sync(ctx context.Context, in *protogen.SyncRequest) (protogen.Extension_SyncClient, error)
	Delete(ctx context.Context, in *protogen.SyncRequest) (protogen.Extension_SyncClient, error)
	Documentation(ctx context.Context) (*protogen.DocumentationResponse, error)
}

type ProtoExtension struct {
	suffiksv1.Extension

	sourceSpec []string
	client     protogen.ExtensionClient
	gclient    *grpc.ClientConn
}

func (e *ProtoExtension) Name() string                  { return e.Extension.Name }
func (e *ProtoExtension) Spec() suffiksv1.ExtensionSpec { return e.Extension.Spec }

func (e *ProtoExtension) Client() protogen.ExtensionClient {
	return e.client
}

func (e *ProtoExtension) Close() error {
	return e.gclient.Close()
}

func (e *ProtoExtension) Default(ctx context.Context, in *protogen.SyncRequest) (*protogen.DefaultResponse, error) {
	return e.client.Default(ctx, in)
}

func (e *ProtoExtension) Validate(ctx context.Context, in *protogen.ValidationRequest) (*protogen.ValidationResponse, error) {
	return e.client.Validate(ctx, in)
}

func (e *ProtoExtension) Sync(ctx context.Context, in *protogen.SyncRequest) (protogen.Extension_SyncClient, error) {
	return e.client.Sync(ctx, in)
}

func (e *ProtoExtension) Delete(ctx context.Context, in *protogen.SyncRequest) (protogen.Extension_SyncClient, error) {
	return e.client.Delete(ctx, in)
}

func (e *ProtoExtension) Documentation(ctx context.Context) (*protogen.DocumentationResponse, error) {
	return e.client.Documentation(ctx, &protogen.DocumentationRequest{})
}

func (e *ProtoExtension) RootKeys() []string {
	return e.sourceSpec
}
