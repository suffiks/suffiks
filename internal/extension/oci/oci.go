package oci

import (
	"context"
	"encoding/json"
	"fmt"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/registry/remote"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	MediaTypeWASI = "application/vnd.com.suffiks.wasi.v1"
	MediaTypeDocs = "application/vnd.com.suffiks.docs.layer.v1+tar"
)

func Get(ctx context.Context, image, tag string) (map[string][]byte, error) {
	store := memory.New()

	// 1. Connect to a remote repository
	repo, err := remote.NewRepository(image)
	if err != nil {
		return nil, fmt.Errorf("oci.Get: unable to create remote repository: %w", err)
	}

	// 3. Copy from the remote repository to the OCI layout store
	manifestDescriptor, err := oras.Copy(ctx, repo, tag, store, tag, oras.DefaultCopyOptions)
	if err != nil {
		return nil, fmt.Errorf("oci.Get: unable to copy from remote repository: %w", err)
	}

	// 3. Fetch from OCI layout store to verify
	fetched, err := content.FetchAll(ctx, store, manifestDescriptor)
	if err != nil {
		return nil, fmt.Errorf("oci.Get: unable to fetch from OCI registry: %w", err)
	}

	var manifest v1.Manifest
	if err := json.Unmarshal(fetched, &manifest); err != nil {
		return nil, fmt.Errorf("oci.Get: unable to unmarshal manifest: %w", err)
	}

	files := map[string][]byte{}
	for _, layer := range manifest.Layers {
		switch layer.MediaType {
		case MediaTypeWASI, MediaTypeDocs:
			b, err := content.FetchAll(ctx, store, layer)
			if err != nil {
				return nil, fmt.Errorf("oci.Get: unable to fetch layer: %w", err)
			}

			files[layer.MediaType] = b
		default:
			log.FromContext(ctx, "oci.Get").Info("ignoring layer", "mediaType", layer.MediaType)
		}
	}

	return files, nil
}
