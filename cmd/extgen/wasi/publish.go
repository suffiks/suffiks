package wasi

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/suffiks/suffiks/internal/extension/oci"
	"github.com/urfave/cli/v2"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	credentials "oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"
)

func publish() *cli.Command {
	return &cli.Command{
		Name:        "publish",
		Description: "Publish a wasi file to a registry",
		ArgsUsage:   "A single argument is required: path to the wasi file",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:      "docs",
				Usage:     "Path to docs directory",
				TakesFile: true,
			},
			&cli.StringFlag{
				Name:     "tag",
				Usage:    "Name and optionally a tag in the `name:tag` format",
				Required: true,
			},
		},
		Before: func(c *cli.Context) error {
			if c.NArg() != 1 {
				return cli.Exit("path to wasi file is required", 1)
			}

			if s, err := os.Stat(c.Args().First()); err != nil {
				return err
			} else if s.IsDir() {
				return cli.Exit("path to wasi file is required", 1)
			}

			if c.String("docs") != "" {
				if s, err := os.Stat(c.String("docs")); err != nil {
					return cli.Exit("error when opening docs directory: %w", 1)
				} else if !s.IsDir() {
					return cli.Exit("--dir must be a directory", 1)
				}
			}
			return nil
		},
		Action: func(c *cli.Context) error {
			ctx := c.Context

			fs := memory.New()

			name, tag, err := parseTag(c.String("tag"))
			if err != nil {
				return err
			}

			b, err := os.ReadFile(c.Args().First())
			if err != nil {
				return err
			}

			layers := []v1.Descriptor{}
			wasiDesc, err := pushBlob(ctx, oci.MediaTypeWASI, b, fs)
			if err != nil {
				return err
			}
			layers = append(layers, wasiDesc)

			if c.String("docs") != "" {
				buf := &bytes.Buffer{}
				zw := gzip.NewWriter(buf)
				wr := tar.NewWriter(zw)

				err = filepath.Walk(c.String("docs"), func(path string, info os.FileInfo, err error) error {
					if err != nil || filepath.Ext(path) != ".md" {
						return nil
					}

					header, err := tar.FileInfoHeader(info, path)
					if err != nil {
						return err
					}

					header.Name = filepath.ToSlash(path)
					if err := wr.WriteHeader(header); err != nil {
						return err
					}

					if info.IsDir() {
						return nil
					}

					f, err := os.Open(path)
					if err != nil {
						return err
					}
					defer f.Close()

					if _, err := io.Copy(wr, f); err != nil {
						return err
					}

					return nil
				})
				zw.Close()
				wr.Close()
				if err != nil {
					return err
				}

				docsDesc, err := pushBlob(ctx, oci.MediaTypeDocs, buf.Bytes(), fs)
				if err != nil {
					return err
				}
				layers = append(layers, docsDesc)
			}

			configBlob := []byte("Hello config")
			configDesc, err := pushBlob(ctx, v1.MediaTypeImageConfig, configBlob, fs)
			if err != nil {
				panic(err)
			}

			content := v1.Manifest{
				Config:    configDesc,
				Layers:    layers,
				Versioned: specs.Versioned{SchemaVersion: 2},
			}
			contentBytes, err := json.Marshal(content)
			if err != nil {
				panic(err)
			}

			manifestDesc, err := pushBlob(ctx, v1.MediaTypeImageManifest, contentBytes, fs)
			if err != nil {
				panic(err)
			}

			if err = fs.Tag(ctx, manifestDesc, tag); err != nil {
				return err
			}

			storeOpts := credentials.StoreOptions{}
			credStore, err := credentials.NewStoreFromDocker(storeOpts)
			if err != nil {
				return err
			}

			// 3.2. Connect to a remote repository
			repo, err := remote.NewRepository(name)
			if err != nil {
				panic(err)
			}

			// Prepare the auth client for the registry and credential store
			repo.Client = &auth.Client{
				Client:     retry.DefaultClient,
				Cache:      auth.DefaultCache,
				Credential: credentials.Credential(credStore), // Use the credential store
			}

			// 3. Copy from the file store to the remote repository
			_, err = oras.Copy(ctx, fs, tag, repo, tag, oras.DefaultCopyOptions)
			if err != nil {
				return fmt.Errorf("failed to copy: %w", err)
			}

			return nil
		},
	}
}

func pushBlob(ctx context.Context, mediaType string, blob []byte, target oras.Target) (desc v1.Descriptor, err error) {
	desc = v1.Descriptor{
		MediaType: mediaType,
		Digest:    digest.FromBytes(blob),
		Size:      int64(len(blob)),
	}
	return desc, target.Push(ctx, desc, bytes.NewReader(blob))
}

// Parse tag into name and reference, with default reference "latest"
func parseTag(tag string) (name, ref string, err error) {
	parts := strings.Split(tag, ":")
	if len(parts) == 1 {
		return parts[0], "latest", nil
	}
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid tag %q, expected name:tag", tag)
	}
	return parts[0], parts[1], nil
}
