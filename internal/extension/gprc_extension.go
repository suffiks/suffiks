package extension

import (
	"context"
	"encoding/json"

	suffiksv1 "github.com/suffiks/suffiks/api/suffiks/v1"
	"github.com/suffiks/suffiks/extension/protogen"
	"google.golang.org/grpc"
)

type GRPC struct {
	suffiksv1.Extension

	sourceSpec []string
	client     protogen.ExtensionClient
	gclient    *grpc.ClientConn
}

func (g *GRPC) Name() string                  { return g.Extension.Name }
func (g *GRPC) Spec() suffiksv1.ExtensionSpec { return g.Extension.Spec }
func (g *GRPC) Close(context.Context) error   { return g.gclient.Close() }

func (g *GRPC) Default(ctx context.Context, in *protogen.SyncRequest) (*protogen.DefaultResponse, error) {
	return g.client.Default(ctx, in)
}

func (g *GRPC) Validate(ctx context.Context, in *protogen.ValidationRequest) (*protogen.ValidationResponse, error) {
	return g.client.Validate(ctx, in)
}

func (g *GRPC) Sync(ctx context.Context, in *protogen.SyncRequest) (StreamResponse, error) {
	return g.client.Sync(ctx, in)
}

func (g *GRPC) Delete(ctx context.Context, in *protogen.SyncRequest) (StreamResponse, error) {
	return g.client.Delete(ctx, in)
}

func (g *GRPC) Documentation(ctx context.Context) (*protogen.DocumentationResponse, error) {
	return g.client.Documentation(ctx, &protogen.DocumentationRequest{})
}

func (g *GRPC) RootKeys() []string {
	return g.sourceSpec
}

func (g *GRPC) init() error {
	props := &properties{}

	if err := json.Unmarshal(g.Spec().OpenAPIV3Schema.Raw, props); err != nil {
		return err
	}
	for key := range props.Properties {
		g.sourceSpec = append(g.sourceSpec, key)
	}
	return nil
}
