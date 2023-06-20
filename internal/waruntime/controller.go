package waruntime

import (
	"context"
	"fmt"
	"sync"

	suffiksv1 "github.com/suffiks/suffiks/pkg/api/suffiks/v1"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"k8s.io/client-go/dynamic"
)

type extension struct {
	version            string
	module             wazero.CompiledModule
	clientPermissions  map[string]struct{}
	configMapReference *suffiksv1.ConfigMapReference
}

type Controller struct {
	cache wazero.CompilationCache

	lock       sync.RWMutex
	extensions map[string]extension
}

func New(ctx context.Context) *Controller {
	cache := wazero.NewCompilationCache()

	return &Controller{
		cache:      cache,
		extensions: make(map[string]extension),
	}
}

func (c *Controller) NewRunner(ctx context.Context, extension string, client dynamic.Interface) (*Runner, error) {
	ext, ok := c.getModule(extension)

	if !ok {
		return nil, fmt.Errorf("%w: %v", ErrExtensionNotFound, extension)
	}

	runtimeConfig := wazero.NewRuntimeConfig().WithCompilationCache(c.cache).WithCoreFeatures(api.CoreFeaturesV2)
	r := wazero.NewRuntimeWithConfig(ctx, runtimeConfig)
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	return &Runner{
		name:               extension,
		version:            ext.version,
		controller:         c,
		module:             ext.module,
		runtime:            r,
		client:             client,
		clientPermissions:  ext.clientPermissions,
		configMapReference: ext.configMapReference,
	}, nil
}

func (c *Controller) Close(ctx context.Context) error {
	return c.cache.Close(ctx)
}

func (c *Controller) getModule(name string) (ext extension, ok bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	ext, ok = c.extensions[name]
	return ext, ok
}

func (c *Controller) Load(ctx context.Context, name, version string, module []byte, clientPermissions map[string]struct{}, configMapReference *suffiksv1.ConfigMapReference) error {
	ext, ok := c.getModule(name)
	if ok && ext.version == version {
		return nil
	}

	runtimeConfig := wazero.NewRuntimeConfig().WithCompilationCache(c.cache).WithCoreFeatures(api.CoreFeaturesV2)
	r := wazero.NewRuntimeWithConfig(ctx, runtimeConfig)
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	cm, err := r.CompileModule(ctx, module)
	if err != nil {
		return err
	}

	if err := validate(cm); err != nil {
		cm.Close(ctx)
		return err
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	if ok {
		if err := c.extensions[name].module.Close(ctx); err != nil {
			return err
		}
	}

	c.extensions[name] = extension{
		version:            version,
		module:             cm,
		clientPermissions:  clientPermissions,
		configMapReference: configMapReference,
	}

	return nil
}
