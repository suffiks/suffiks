package waruntime

import (
	"context"
	"fmt"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"k8s.io/client-go/dynamic"
)

type extension struct {
	version string
	module  wazero.CompiledModule
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
	m, _, ok := c.getModule(extension)

	if !ok {
		return nil, fmt.Errorf("%w: %v", ErrExtensionNotFound, extension)
	}

	runtimeConfig := wazero.NewRuntimeConfig().WithCompilationCache(c.cache).WithCoreFeatures(api.CoreFeaturesV2)
	r := wazero.NewRuntimeWithConfig(ctx, runtimeConfig)
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	return &Runner{
		controller: c,
		module:     m,
		runtime:    r,
		client:     client,
	}, nil
}

func (c *Controller) Close(ctx context.Context) error {
	return c.cache.Close(ctx)
}

func (c *Controller) getModule(name string) (module wazero.CompiledModule, version string, ok bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	ext, ok := c.extensions[name]
	if !ok {
		return nil, "", false
	}
	return ext.module, ext.version, ok
}

func (c *Controller) Load(ctx context.Context, name, version string, module []byte) error {
	_, mver, ok := c.getModule(name)
	if ok && mver == version {
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
		version: version,
		module:  cm,
	}

	return nil
}
