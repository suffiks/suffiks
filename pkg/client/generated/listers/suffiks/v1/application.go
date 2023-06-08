// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/suffiks/suffiks/api/suffiks/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ApplicationLister helps list Applications.
// All objects returned here must be treated as read-only.
type ApplicationLister interface {
	// List lists all Applications in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.Application, err error)
	// Applications returns an object that can list and get Applications.
	Applications(namespace string) ApplicationNamespaceLister
	ApplicationListerExpansion
}

// applicationLister implements the ApplicationLister interface.
type applicationLister struct {
	indexer cache.Indexer
}

// NewApplicationLister returns a new ApplicationLister.
func NewApplicationLister(indexer cache.Indexer) ApplicationLister {
	return &applicationLister{indexer: indexer}
}

// List lists all Applications in the indexer.
func (s *applicationLister) List(selector labels.Selector) (ret []*v1.Application, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.Application))
	})
	return ret, err
}

// Applications returns an object that can list and get Applications.
func (s *applicationLister) Applications(namespace string) ApplicationNamespaceLister {
	return applicationNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// ApplicationNamespaceLister helps list and get Applications.
// All objects returned here must be treated as read-only.
type ApplicationNamespaceLister interface {
	// List lists all Applications in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.Application, err error)
	// Get retrieves the Application from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.Application, error)
	ApplicationNamespaceListerExpansion
}

// applicationNamespaceLister implements the ApplicationNamespaceLister
// interface.
type applicationNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Applications in the indexer for a given namespace.
func (s applicationNamespaceLister) List(selector labels.Selector) (ret []*v1.Application, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.Application))
	})
	return ret, err
}

// Get retrieves the Application from the indexer for a given namespace and name.
func (s applicationNamespaceLister) Get(name string) (*v1.Application, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("application"), name)
	}
	return obj.(*v1.Application), nil
}
