package extension

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	suffiksv1 "github.com/suffiks/suffiks/api/suffiks/v1"
	"github.com/suffiks/suffiks/extension/protogen"
	"github.com/suffiks/suffiks/internal/extension/oci"
	"github.com/suffiks/suffiks/internal/waruntime"
)

type WASI struct {
	suffiksv1.Extension

	controller *waruntime.Controller
	pages      [][]byte

	sourceSpec []string
	instance   *waruntime.Runner
}

func NewWASI(ext suffiksv1.Extension) (*WASI, error) {
	return &WASI{
		Extension: ext,
	}, nil
}

func (w *WASI) Name() string                  { return w.Extension.Name }
func (w *WASI) Spec() suffiksv1.ExtensionSpec { return w.Extension.Spec }
func (w *WASI) RootKeys() []string            { return w.sourceSpec }

func (w *WASI) Close(ctx context.Context) error {
	return w.instance.Close(ctx)
}

func (w *WASI) Default(ctx context.Context, in *protogen.SyncRequest) (*protogen.DefaultResponse, error) {
	return w.instance.Defaulting(ctx, in)
}

func (w *WASI) Validate(ctx context.Context, in *protogen.ValidationRequest) (*protogen.ValidationResponse, error) {
	errs, err := w.instance.Validate(ctx, in)
	if err != nil {
		return nil, err
	}

	return &protogen.ValidationResponse{
		Errors: errs,
	}, nil
}

func (w *WASI) Sync(ctx context.Context, in *protogen.SyncRequest) (StreamResponse, error) {
	return w.instance.Sync(ctx, in)
}

func (w *WASI) Delete(ctx context.Context, in *protogen.SyncRequest) (StreamResponse, error) {
	return nil, w.instance.Delete(ctx, in)
}

func (w *WASI) Documentation(ctx context.Context) (*protogen.DocumentationResponse, error) {
	return &protogen.DocumentationResponse{
		Pages: w.pages,
	}, nil
}

func (w *WASI) init(files map[string][]byte) error {
	props := &properties{}
	if err := json.Unmarshal(w.Spec().OpenAPIV3Schema.Raw, props); err != nil {
		return err
	}
	for key := range props.Properties {
		w.sourceSpec = append(w.sourceSpec, key)
	}

	err := w.controller.Load(context.Background(), w.Name(), w.Spec().Controller.WASI.ImageTag(), files[oci.MediaTypeWASI])
	if err != nil {
		return fmt.Errorf("WASI.init: error loading wasi module: %w", err)
	}

	if err := w.initDocs(files); err != nil {
		return fmt.Errorf("WASI.init: error loading docs: %w", err)
	}
	return nil
}

func (w *WASI) initDocs(files map[string][]byte) error {
	b, ok := files[oci.MediaTypeDocs]
	if !ok {
		return nil
	}

	// untar gzipped tarball
	gzipReader, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return err
	}

	r := tar.NewReader(gzipReader)
	for {
		hdr, err := r.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if filepath.Ext(hdr.Name) != ".md" {
			continue
		}

		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(r); err != nil {
			return err
		}

		w.pages = append(w.pages, buf.Bytes())
	}

	return nil
}
