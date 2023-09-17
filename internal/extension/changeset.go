package extension

import (
	"encoding/json"
	"fmt"
	"sync"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/suffiks/suffiks/extension/protogen"
	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Changeset struct {
	lock sync.Mutex

	environment []v1.EnvVar
	labels      map[string]string
	annotations map[string]string
	envFrom     []v1.EnvFromSource
	mergePatch  []byte
}

func (c *Changeset) Add(resp *protogen.Response) error {
	if r, ok := resp.OFResponse.(*protogen.Response_MergePatch); ok {
		return c.AddMergePatch(r.MergePatch)
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	switch r := resp.OFResponse.(type) {
	case *protogen.Response_Env:
		c.environment = append(c.environment, v1.EnvVar{
			Name:  r.Env.Name,
			Value: r.Env.Value,
		})
	case *protogen.Response_Annotation:
		if c.annotations == nil {
			c.annotations = map[string]string{}
		}
		c.annotations[r.Annotation.Name] = r.Annotation.Value
	case *protogen.Response_Label:
		if c.labels == nil {
			c.labels = map[string]string{}
		}
		c.labels[r.Label.Name] = r.Label.Value
	case *protogen.Response_EnvFrom:
		switch r.EnvFrom.GetType() {
		case protogen.EnvFromType_SECRET:
			c.envFrom = append(c.envFrom, v1.EnvFromSource{
				SecretRef: &v1.SecretEnvSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: r.EnvFrom.Name,
					},
					Optional: ptr.To(r.EnvFrom.Optional),
				},
			})
		case protogen.EnvFromType_CONFIGMAP:
			c.envFrom = append(c.envFrom, v1.EnvFromSource{
				ConfigMapRef: &v1.ConfigMapEnvSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: r.EnvFrom.Name,
					},
					Optional: ptr.To(r.EnvFrom.Optional),
				},
			})
		default:
			return fmt.Errorf("unknown envfrom type: %q", r.EnvFrom.GetType())
		}
	default:
		return fmt.Errorf("unexpected response type: %T", r)
	}
	return nil
}

func (changeset *Changeset) Apply(v client.Object) error {
	v.SetLabels(mergeMaps(v.GetLabels(), changeset.labels))
	v.SetAnnotations(mergeMaps(v.GetAnnotations(), changeset.annotations))

	if len(changeset.mergePatch) > 0 {
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("applyChangeset unmarshal: %w", err)
		}
		out, err := jsonpatch.MergePatch(b, changeset.mergePatch)
		if err != nil {
			return fmt.Errorf("applyChangeset mergePatch: %w", err)
		}
		err = json.Unmarshal(out, v)
		if err != nil {
			return fmt.Errorf("applyChangeset unmarshal: %w", err)
		}
	}
	return nil
}

func (c *Changeset) AddMergePatch(patch []byte) error {
	if len(patch) == 0 {
		return nil
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	if len(c.mergePatch) == 0 {
		c.mergePatch = patch
	} else {
		combined, err := jsonpatch.MergeMergePatches(c.mergePatch, patch)
		if err != nil {
			return err
		}
		c.mergePatch = combined
	}

	return nil
}

func mergeMaps(maps ...map[string]string) map[string]string {
	ret := map[string]string{}
	for _, m := range maps {
		for k, v := range m {
			ret[k] = v
		}
	}
	return ret
}
