package base

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"

	jsonpatch "github.com/evanphx/json-patch/v5"
	suffiksv1 "github.com/suffiks/suffiks/apis/suffiks/v1"
	"github.com/suffiks/suffiks/base/runtime"
	"github.com/suffiks/suffiks/extension/protogen"
	"google.golang.org/grpc"
	v1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

type keyValue map[string]any

type Changeset struct {
	lock sync.Mutex

	Environment []v1.EnvVar
	Labels      map[string]string
	Annotations map[string]string
	EnvFrom     []v1.EnvFromSource
	MergePatch  []byte
}

func (c *Changeset) AddMergePatch(patch []byte) error {
	if len(patch) == 0 {
		return nil
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	if len(c.MergePatch) == 0 {
		c.MergePatch = patch
	} else {
		combined, err := jsonpatch.MergeMergePatches(c.MergePatch, patch)
		if err != nil {
			return err
		}
		c.MergePatch = combined
	}
	return nil
}

type ExtensionManager struct {
	grpcOptions []grpc.DialOption

	lock sync.Mutex
	spec map[suffiksv1.Target]*runtime.Generator

	rwlock     sync.RWMutex
	extensions map[string]extension
}

// NewExtensionManager creates a new ExtensionManager. It reads all .yaml files from the provided fs.FS as base types.
func NewExtensionManager(files fs.FS, grpcOptions []grpc.DialOption) (*ExtensionManager, error) {
	mgr := &ExtensionManager{
		grpcOptions: grpcOptions,

		spec:       map[suffiksv1.Target]*runtime.Generator{},
		extensions: map[string]extension{},
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

		crd, err := runtime.FromYAML(file)
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
	spec := ext.Spec.OpenAPIV3Schema.Raw

	gclient, err := grpc.Dial(ext.Spec.Controller.Target(), c.grpcOptions...)
	if err != nil {
		return fmt.Errorf("ExtensionManager.add: grpc dial error: %w", err)
	}

	client := protogen.NewExtensionClient(gclient)

	c.lock.Lock()
	defer c.lock.Unlock()
	g, ok := c.spec[target]
	if !ok {
		return fmt.Errorf("%q not a valid target", target)
	}
	if err := g.Add(spec); err != nil {
		return err
	}

	c.rwlock.Lock()
	defer c.rwlock.Unlock()

	wext := extension{
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

func (c *ExtensionManager) Remove(ext *suffiksv1.Extension) error {
	c.lock.Lock()
	defer c.lock.Unlock()

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
	c.lock.Lock()
	defer c.lock.Unlock()

	g, ok := c.spec[target]
	if !ok {
		return nil
	}
	return g.Schema()
}

func (c *ExtensionManager) ExtensionsFor(kind string) []extension {
	c.rwlock.RLock()
	defer c.rwlock.RUnlock()

	cp := []extension{}
	for _, v := range c.extensions {
		if contains(v.Spec.Targets, suffiksv1.Target(kind)) {
			cp = append(cp, v)
		}
	}

	return cp
}

func (c *ExtensionManager) All() []extension {
	c.rwlock.RLock()
	defer c.rwlock.RUnlock()

	cp := []extension{}
	added := map[string]struct{}{}
	for _, v := range c.extensions {
		if _, ok := added[v.Name]; ok {
			continue
		}
		added[v.Name] = struct{}{}
		cp = append(cp, v)
	}

	return cp
}

type extension struct {
	suffiksv1.Extension

	sourceSpec []string
	client     protogen.ExtensionClient
	gclient    *grpc.ClientConn
}

func (e *extension) Client() protogen.ExtensionClient {
	return e.client
}

type properties struct {
	Properties map[string]any `json:"properties"`
}

func (e *extension) init() error {
	props := &properties{}

	if err := json.Unmarshal(e.Spec.OpenAPIV3Schema.Raw, props); err != nil {
		return err
	}
	for key := range props.Properties {
		e.sourceSpec = append(e.sourceSpec, key)
	}
	return nil
}
