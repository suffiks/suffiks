package extension

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"time"

	suffiksv1 "github.com/suffiks/suffiks/api/suffiks/v1"
	"github.com/suffiks/suffiks/extension/protogen"
	"github.com/suffiks/suffiks/internal/extension/oci"
	"github.com/suffiks/suffiks/internal/specgen"
	"github.com/suffiks/suffiks/internal/waruntime"
	"google.golang.org/grpc"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/dynamic"
)

type KeyValue map[string]any

type Option func(*ExtensionManager)

type WASILoader func(ctx context.Context, image, tag string) (map[string][]byte, error)

func WithWASILoader(loader WASILoader) Option {
	return func(mgr *ExtensionManager) {
		mgr.wasiLoader = loader
	}
}

func WithGRPCOptions(opts ...grpc.DialOption) Option {
	return func(mgr *ExtensionManager) {
		mgr.grpcOptions = opts
	}
}

type ExtensionManager struct {
	grpcOptions    []grpc.DialOption
	wasiController *waruntime.Controller
	wasiLoader     WASILoader
	dynamicClient  dynamic.Interface

	specLock sync.Mutex
	spec     map[suffiksv1.Target]*specgen.Generator

	rwlock     sync.RWMutex
	extensions map[string]Extension
}

// NewExtensionManager creates a new ExtensionManager. It reads all .yaml files from the provided fs.FS as base types.
func NewExtensionManager(ctx context.Context, files fs.FS, dynClient dynamic.Interface, opts ...Option) (*ExtensionManager, error) {
	mgr := &ExtensionManager{
		wasiController: waruntime.New(ctx),
		wasiLoader:     oci.Get,
		dynamicClient:  dynClient,

		spec:       map[suffiksv1.Target]*specgen.Generator{},
		extensions: map[string]Extension{},
	}

	for _, opt := range opts {
		opt(mgr)
	}

	err := fs.WalkDir(files, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) != ".yaml" {
			return nil
		}
		file, err := files.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		crd, err := specgen.FromYAML(file)
		if err != nil {
			return err
		}
		mgr.spec[suffiksv1.Target(strings.TrimPrefix(crd.Kind(), "Base"))] = crd

		return nil
	})

	return mgr, err
}

func (c *ExtensionManager) Add(ext suffiksv1.Extension) error {
	for _, target := range ext.Spec.Targets {
		if err := c.add(ext, target); err != nil {
			return err
		}
	}
	return nil
}

func (c *ExtensionManager) add(ext suffiksv1.Extension, target suffiksv1.Target) error {
	if ext.Spec.Controller.GRPC != nil {
		return c.addGRPC(ext, target)
	} else if ext.Spec.Controller.WASI != nil {
		return c.addWASI(ext, target)
	}

	return fmt.Errorf("ExtensionManager.add: no controller specified")
}

func (c *ExtensionManager) addGRPC(ext suffiksv1.Extension, target suffiksv1.Target) error {
	gclient, err := grpc.Dial(ext.Spec.Controller.GRPC.Target(), c.grpcOptions...)
	if err != nil {
		return fmt.Errorf("ExtensionManager.add: grpc dial error: %w", err)
	}

	client := protogen.NewExtensionClient(gclient)

	c.specLock.Lock()
	defer c.specLock.Unlock()
	g, ok := c.spec[target]
	if !ok {
		return fmt.Errorf("%q not a valid target", target)
	}

	spec := ext.Spec.OpenAPIV3Schema.Raw
	if err := g.Add(spec); err != nil {
		return err
	}

	c.rwlock.Lock()
	defer c.rwlock.Unlock()

	wext := &GRPC{
		Extension: ext,
		client:    client,
		gclient:   gclient,
	}
	if err := wext.init(); err != nil {
		return err
	}
	c.extensions[ext.Name] = wext

	return nil
}

func (c *ExtensionManager) addWASI(ext suffiksv1.Extension, target suffiksv1.Target) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	files, err := c.wasiLoader(ctx, ext.Spec.Controller.WASI.Image, ext.Spec.Controller.WASI.Tag)
	if err != nil {
		return fmt.Errorf("ExtensionManager.add: oci get error: %w", err)
	}

	c.specLock.Lock()
	defer c.specLock.Unlock()
	g, ok := c.spec[target]
	if !ok {
		return fmt.Errorf("%q not a valid target", target)
	}

	spec := ext.Spec.OpenAPIV3Schema.Raw
	if err := g.Add(spec); err != nil {
		return err
	}

	c.rwlock.Lock()
	defer c.rwlock.Unlock()

	wext := NewWASI(
		ext,
		c.wasiController,
		c.dynamicClient,
	)
	if err := wext.init(files); err != nil {
		return err
	}
	c.extensions[ext.Name] = wext

	return nil
}

func (c *ExtensionManager) Remove(ext *suffiksv1.Extension) error {
	c.specLock.Lock()
	defer c.specLock.Unlock()

	for _, target := range ext.Spec.Targets {
		g, ok := c.spec[target]
		if !ok {
			return fmt.Errorf("%q not a valid target", target)
		}
		if err := g.Remove(ext.Spec.OpenAPIV3Schema.Raw); err != nil {
			return err
		}
	}

	delete(c.extensions, ext.Name)

	return nil
}

func (c *ExtensionManager) Schema(target suffiksv1.Target) *apiextv1.JSONSchemaProps {
	c.specLock.Lock()
	defer c.specLock.Unlock()

	g, ok := c.spec[target]
	if !ok {
		return nil
	}
	return g.Schema()
}

func (c *ExtensionManager) ExtensionsFor(kind string) []Extension {
	c.rwlock.RLock()
	defer c.rwlock.RUnlock()

	cp := []Extension{}
	for _, v := range c.extensions {
		if contains(v.Spec().Targets, suffiksv1.Target(kind)) {
			cp = append(cp, v)
		}
	}

	return cp
}

func (c *ExtensionManager) All() []Extension {
	c.rwlock.RLock()
	defer c.rwlock.RUnlock()

	cp := []Extension{}
	added := map[string]struct{}{}
	for _, v := range c.extensions {
		if _, ok := added[v.Name()]; ok {
			continue
		}
		added[v.Name()] = struct{}{}
		cp = append(cp, v)
	}

	return cp
}

type properties struct {
	Properties map[string]any `json:"properties"`
}

func contains[T comparable](arr []T, val T) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}
