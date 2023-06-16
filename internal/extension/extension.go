package extension

import (
	"context"

	"github.com/suffiks/suffiks/extension/protogen"
	suffiksv1 "github.com/suffiks/suffiks/pkg/api/suffiks/v1"
)

type StreamResponse interface {
	Recv() (*protogen.Response, error)
}

type Extension interface {
	Name() string
	Spec() suffiksv1.ExtensionSpec
	RootKeys() []string
	Close(context.Context) error

	Default(ctx context.Context, in *protogen.SyncRequest) (*protogen.DefaultResponse, error)
	Validate(ctx context.Context, in *protogen.ValidationRequest) (*protogen.ValidationResponse, error)
	Sync(ctx context.Context, in *protogen.SyncRequest) (StreamResponse, error)
	Delete(ctx context.Context, in *protogen.SyncRequest) (StreamResponse, error)
	Documentation(ctx context.Context) (*protogen.DocumentationResponse, error)
}
