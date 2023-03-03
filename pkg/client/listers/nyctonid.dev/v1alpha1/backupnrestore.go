/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/nidhey27/backup-and-restore/pkg/apis/nyctonid.dev/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// BackupNRestoreLister helps list BackupNRestores.
// All objects returned here must be treated as read-only.
type BackupNRestoreLister interface {
	// List lists all BackupNRestores in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.BackupNRestore, err error)
	// BackupNRestores returns an object that can list and get BackupNRestores.
	BackupNRestores(namespace string) BackupNRestoreNamespaceLister
	BackupNRestoreListerExpansion
}

// backupNRestoreLister implements the BackupNRestoreLister interface.
type backupNRestoreLister struct {
	indexer cache.Indexer
}

// NewBackupNRestoreLister returns a new BackupNRestoreLister.
func NewBackupNRestoreLister(indexer cache.Indexer) BackupNRestoreLister {
	return &backupNRestoreLister{indexer: indexer}
}

// List lists all BackupNRestores in the indexer.
func (s *backupNRestoreLister) List(selector labels.Selector) (ret []*v1alpha1.BackupNRestore, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.BackupNRestore))
	})
	return ret, err
}

// BackupNRestores returns an object that can list and get BackupNRestores.
func (s *backupNRestoreLister) BackupNRestores(namespace string) BackupNRestoreNamespaceLister {
	return backupNRestoreNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// BackupNRestoreNamespaceLister helps list and get BackupNRestores.
// All objects returned here must be treated as read-only.
type BackupNRestoreNamespaceLister interface {
	// List lists all BackupNRestores in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.BackupNRestore, err error)
	// Get retrieves the BackupNRestore from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.BackupNRestore, error)
	BackupNRestoreNamespaceListerExpansion
}

// backupNRestoreNamespaceLister implements the BackupNRestoreNamespaceLister
// interface.
type backupNRestoreNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all BackupNRestores in the indexer for a given namespace.
func (s backupNRestoreNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.BackupNRestore, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.BackupNRestore))
	})
	return ret, err
}

// Get retrieves the BackupNRestore from the indexer for a given namespace and name.
func (s backupNRestoreNamespaceLister) Get(name string) (*v1alpha1.BackupNRestore, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("backupnrestore"), name)
	}
	return obj.(*v1alpha1.BackupNRestore), nil
}
