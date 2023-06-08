package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/suffiks/suffiks/base/tracing"
	"github.com/suffiks/suffiks/extension/protogen"
	"github.com/suffiks/suffiks/internal/extension"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type lockedList[T comparable] struct {
	lock sync.Mutex
	list []T
}

func (l *lockedList[T]) Add(val T) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.list = append(l.list, val)
}

func (l *lockedList[T]) Contains(val T) bool {
	l.lock.Lock()
	defer l.lock.Unlock()
	return contains(l.list, val)
}

func (l *lockedList[T]) Slice() []T {
	return l.list[:]
}

type Result struct {
	Changeset *extension.Changeset

	// Extensions contains the name of extensions that were ran during the operation.
	Extensions lockedList[string]
}

type ExtManager interface {
	ExtensionsFor(kind string) []extension.Extension
}

type Object interface {
	client.Object

	GetSpec() []byte
}

type responder interface {
	Recv() (*protogen.Response, error)
}

type (
	requestFunc   func(ctx context.Context, ext extension.Extension, in *protogen.SyncRequest) (responder, error)
	shouldRunFunc func(e extension.Extension, cu *protogen.SyncRequest) bool
)

type FieldErrsWrapper field.ErrorList

func (f FieldErrsWrapper) Error() string {
	return "Field errors"
}

type ExtensionController struct {
	manager ExtManager
	metrics *prometheus.HistogramVec
}

func NewExtensionController(manager ExtManager) *ExtensionController {
	return &ExtensionController{
		manager: manager,
		metrics: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "suffiks_extension_manager_duration_seconds",
			Help:    "Duration of extension manager operations",
			Buckets: []float64{.005, .01, .05, .1, .25, .5, 1, 2.5, 5, 10},
		}, []string{"operation", "extension", "status"}),
	}
}

func (c *ExtensionController) RegisterMetrics(reg prometheus.Registerer) error {
	return reg.Register(c.metrics)
}

func (c *ExtensionController) Sync(ctx context.Context, v Object) (*Result, error) {
	f := func(ctx context.Context, ext extension.Extension, in *protogen.SyncRequest) (responder, error) {
		return ext.Sync(ctx, in)
	}

	return c.run(ctx, "sync", v, f, func(e extension.Extension, cu *protogen.SyncRequest) bool {
		return e.Spec().Always || len(cu.Spec) > 0
	})
}

func (c *ExtensionController) Delete(ctx context.Context, v Object) error {
	f := func(ctx context.Context, ext extension.Extension, in *protogen.SyncRequest) (responder, error) {
		return ext.Delete(ctx, in)
	}

	_, err := c.run(ctx, "delete", v, f, func(e extension.Extension, cu *protogen.SyncRequest) bool {
		return e.Spec().Always || len(cu.Spec) > 0
	})
	return err
}

func (c *ExtensionController) DeleteExtension(ctx context.Context, v Object, extensionName string) error {
	f := func(ctx context.Context, ext extension.Extension, in *protogen.SyncRequest) (responder, error) {
		return ext.Delete(ctx, in)
	}

	_, err := c.run(ctx, "delete", v, f, func(e extension.Extension, cu *protogen.SyncRequest) bool {
		return e.Name() == extensionName
	})
	return err
}

func (c *ExtensionController) Default(ctx context.Context, obj Object) ([]*protogen.DefaultResponse, error) {
	ctx, span := tracing.Start(ctx, "extensions.Default")
	defer span.End()

	var (
		errs     MultiError
		lock     sync.Mutex
		wg       sync.WaitGroup
		response []*protogen.DefaultResponse

		v extension.KeyValue
	)

	if err := json.Unmarshal(obj.GetSpec(), &v); err != nil {
		return nil, err
	}

	exts := c.manager.ExtensionsFor(obj.GetObjectKind().GroupVersionKind().Kind)
	for _, ext := range exts {
		if !ext.Spec().Webhooks.Defaulting {
			continue
		}

		span.AddEvent("Default " + ext.Name())
		ext := ext
		wg.Add(1)
		go func() {
			defer wg.Done()

			start := time.Now()
			resp, err := c.defaulter(ctx, ext, obj, v)
			if err != nil {
				c.metrics.WithLabelValues("default", ext.Name(), "failure").Observe(time.Since(start).Seconds())
				lock.Lock()
				errs = append(errs, err)
				lock.Unlock()
				return
			}
			c.metrics.WithLabelValues("default", ext.Name(), "success").Observe(time.Since(start).Seconds())

			lock.Lock()
			response = append(response, resp)
			lock.Unlock()
		}()
	}

	wg.Wait()
	if len(errs) > 0 {
		return response, errs
	}

	return response, nil
}

func (c *ExtensionController) defaulter(ctx context.Context, ext extension.Extension, obj Object, v extension.KeyValue) (*protogen.DefaultResponse, error) {
	req, err := createOrUpdateRequest(obj, v, ext)
	if err != nil {
		return nil, err
	}

	return ext.Default(ctx, req)
}

func (c *ExtensionController) Validate(ctx context.Context, typ protogen.ValidationType, newObject, oldObject Object) error {
	ctx, span := tracing.Start(ctx, "extensions.Validate")
	defer span.End()

	var (
		allErrs field.ErrorList
		errs    MultiError
		lock    sync.Mutex
		wg      sync.WaitGroup

		newV, oldV extension.KeyValue
		obj        Object
	)

	if newObject != nil && len(newObject.GetSpec()) > 0 {
		obj = newObject
		if err := json.Unmarshal(newObject.GetSpec(), &newV); err != nil {
			return err
		}
	}
	if oldObject != nil && len(oldObject.GetSpec()) > 0 {
		if obj == nil {
			obj = oldObject
		}
		if err := json.Unmarshal(oldObject.GetSpec(), &oldV); err != nil {
			return err
		}
	}

	exts := c.manager.ExtensionsFor(obj.GetObjectKind().GroupVersionKind().Kind)
	for _, ext := range exts {
		if !ext.Spec().Webhooks.Validation {
			continue
		}

		span.AddEvent("Validate " + ext.Name())
		ext := ext
		wg.Add(1)
		go func() {
			defer wg.Done()

			start := time.Now()
			if err := c.validate(ctx, typ, ext, newObject, oldObject, newV, oldV); err != nil {
				if ferr, ok := err.(FieldErrsWrapper); ok {
					lock.Lock()
					allErrs = append(allErrs, ferr...)
					lock.Unlock()
				} else {
					c.metrics.WithLabelValues("validate", ext.Name(), "failure").Observe(time.Since(start).Seconds())
					lock.Lock()
					errs = append(errs, err)
					lock.Unlock()
					return
				}
			}
			c.metrics.WithLabelValues("validate", ext.Name(), "success").Observe(time.Since(start).Seconds())
		}()
	}

	wg.Wait()
	if len(errs) > 0 {
		return errs
	}

	if len(allErrs) == 0 {
		return nil
	}

	return FieldErrsWrapper(allErrs)
}

func (c *ExtensionController) validate(ctx context.Context, typ protogen.ValidationType, ext extension.Extension, newO, oldO Object, newV, oldV extension.KeyValue) error {
	req := &protogen.ValidationRequest{
		Type: typ,
	}

	if newO != nil {
		ur, err := createOrUpdateRequest(newO, newV, ext)
		if err != nil {
			return err
		}
		req.Sync = ur
	}
	if oldO != nil {
		ur, err := createOrUpdateRequest(oldO, oldV, ext)
		if err != nil {
			return err
		}
		req.Old = ur
	}

	if req.Sync == nil && req.Old == nil {
		return nil
	}

	resp, err := ext.Validate(ctx, req)
	if err != nil {
		return err
	}

	var allErrs field.ErrorList
	for _, e := range resp.Errors {
		allErrs = append(
			allErrs,
			field.Invalid(
				field.NewPath("spec", strings.Split(e.Path, ".")...),
				e.Value,
				e.Detail,
			),
		)
	}
	return FieldErrsWrapper(allErrs)
}

func (c *ExtensionController) run(ctx context.Context, operation string, o Object, rf requestFunc, runFunc shouldRunFunc) (*Result, error) {
	exts := c.manager.ExtensionsFor(o.GetObjectKind().GroupVersionKind().Kind)
	result := &Result{
		Changeset: &extension.Changeset{},
	}
	errs := MultiError{}
	wg := sync.WaitGroup{}

	var v extension.KeyValue
	if err := json.Unmarshal(o.GetSpec(), &v); err != nil {
		return nil, err
	}

	for _, ext := range exts {
		ext := ext

		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			if err := c.runExtension(ctx, ext, o, v, result, rf, runFunc); err != nil {
				c.metrics.WithLabelValues(operation, ext.Name(), "failure").Observe(time.Since(start).Seconds())
				errs = append(errs, err)
				return
			}
			c.metrics.WithLabelValues(operation, ext.Name(), "success").Observe(time.Since(start).Seconds())
		}()
	}

	wg.Wait()
	if len(errs) > 0 {
		return result, errs
	}
	return result, nil
}

func (c *ExtensionController) runExtension(ctx context.Context, ext extension.Extension, o Object, v extension.KeyValue, result *Result, rf requestFunc, runFunc shouldRunFunc) (err error) {
	changeset := result.Changeset

	ctx, span := tracing.Start(ctx, "runExtension")
	defer span.End()
	ur, err := createOrUpdateRequest(o, v, ext)
	if err != nil {
		return err
	} else if !runFunc(ext, ur) {
		return nil
	}

	result.Extensions.Add(ext.Name())

	stream, err := rf(ctx, ext, ur)
	if err != nil {
		return err
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			span.RecordError(err)
			return err
		}

		if err := changeset.Add(resp); err != nil {
			return err
		}
	}
}

// MultiError is a slice of errors implementing the error interface. It is used
// by a Gatherer to report multiple errors during MetricFamily gathering.
type MultiError []error

// Error formats the contained errors as a bullet point list, preceded by the
// total number of errors. Note that this results in a multi-line string.
func (errs MultiError) Error() string {
	if len(errs) == 0 {
		return ""
	}
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "%d error(s) occurred:", len(errs))
	for _, err := range errs {
		fmt.Fprintf(buf, "\n* %s", err)
	}
	return buf.String()
}

func createOrUpdateRequest(o Object, v extension.KeyValue, ext extension.Extension) (*protogen.SyncRequest, error) {
	if v == nil {
		return nil, nil
	}

	ok := o.GetObjectKind()
	ur := &protogen.SyncRequest{
		Owner: &protogen.Owner{
			Kind:        ok.GroupVersionKind().Kind,
			Name:        o.GetName(),
			Namespace:   o.GetNamespace(),
			ApiVersion:  ok.GroupVersionKind().Version,
			Uid:         string(o.GetUID()),
			Labels:      o.GetLabels(),
			Annotations: o.GetAnnotations(),
			RevisionID:  fmt.Sprint(o.GetGeneration()),
		},
	}

	data := map[string]any{}
	for key, val := range v {
		if contains(ext.RootKeys(), key) {
			data[key] = val
		}
	}

	if len(data) == 0 {
		return ur, nil
	}

	var err error
	ur.Spec, err = json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return ur, nil
}

func contains[T comparable](arr []T, val T) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}
