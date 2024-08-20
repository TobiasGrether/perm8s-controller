/*
Copyright 2024 Tobias Grether

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

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	perm8sv1alpha1 "perm8s/pkg/apis/perm8s/v1alpha1"
	versioned "perm8s/pkg/generated/clientset/versioned"
	internalinterfaces "perm8s/pkg/generated/informers/externalversions/internalinterfaces"
	v1alpha1 "perm8s/pkg/generated/listers/perm8s/v1alpha1"
	time "time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// SynchronisationSourceInformer provides access to a shared informer and lister for
// SynchronisationSources.
type SynchronisationSourceInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.SynchronisationSourceLister
}

type synchronisationSourceInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewSynchronisationSourceInformer constructs a new informer for SynchronisationSource type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewSynchronisationSourceInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredSynchronisationSourceInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredSynchronisationSourceInformer constructs a new informer for SynchronisationSource type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredSynchronisationSourceInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Perm8sV1alpha1().SynchronisationSources(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Perm8sV1alpha1().SynchronisationSources(namespace).Watch(context.TODO(), options)
			},
		},
		&perm8sv1alpha1.SynchronisationSource{},
		resyncPeriod,
		indexers,
	)
}

func (f *synchronisationSourceInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredSynchronisationSourceInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *synchronisationSourceInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&perm8sv1alpha1.SynchronisationSource{}, f.defaultInformer)
}

func (f *synchronisationSourceInformer) Lister() v1alpha1.SynchronisationSourceLister {
	return v1alpha1.NewSynchronisationSourceLister(f.Informer().GetIndexer())
}
