// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	"context"
	"time"

	v1 "github.com/suffiks/suffiks/apis/suffiks/v1"
	scheme "github.com/suffiks/suffiks/pkg/client/generated/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ExtensionsGetter has a method to return a ExtensionInterface.
// A group's client should implement this interface.
type ExtensionsGetter interface {
	Extensions(namespace string) ExtensionInterface
}

// ExtensionInterface has methods to work with Extension resources.
type ExtensionInterface interface {
	Create(ctx context.Context, extension *v1.Extension, opts metav1.CreateOptions) (*v1.Extension, error)
	Update(ctx context.Context, extension *v1.Extension, opts metav1.UpdateOptions) (*v1.Extension, error)
	UpdateStatus(ctx context.Context, extension *v1.Extension, opts metav1.UpdateOptions) (*v1.Extension, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Extension, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.ExtensionList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Extension, err error)
	ExtensionExpansion
}

// extensions implements ExtensionInterface
type extensions struct {
	client rest.Interface
	ns     string
}

// newExtensions returns a Extensions
func newExtensions(c *SuffiksV1Client, namespace string) *extensions {
	return &extensions{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the extension, and returns the corresponding extension object, and an error if there is any.
func (c *extensions) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.Extension, err error) {
	result = &v1.Extension{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("extensions").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Extensions that match those selectors.
func (c *extensions) List(ctx context.Context, opts metav1.ListOptions) (result *v1.ExtensionList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.ExtensionList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("extensions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested extensions.
func (c *extensions) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("extensions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a extension and creates it.  Returns the server's representation of the extension, and an error, if there is any.
func (c *extensions) Create(ctx context.Context, extension *v1.Extension, opts metav1.CreateOptions) (result *v1.Extension, err error) {
	result = &v1.Extension{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("extensions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(extension).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a extension and updates it. Returns the server's representation of the extension, and an error, if there is any.
func (c *extensions) Update(ctx context.Context, extension *v1.Extension, opts metav1.UpdateOptions) (result *v1.Extension, err error) {
	result = &v1.Extension{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("extensions").
		Name(extension.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(extension).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *extensions) UpdateStatus(ctx context.Context, extension *v1.Extension, opts metav1.UpdateOptions) (result *v1.Extension, err error) {
	result = &v1.Extension{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("extensions").
		Name(extension.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(extension).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the extension and deletes it. Returns an error if one occurs.
func (c *extensions) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("extensions").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *extensions) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("extensions").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched extension.
func (c *extensions) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Extension, err error) {
	result = &v1.Extension{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("extensions").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}