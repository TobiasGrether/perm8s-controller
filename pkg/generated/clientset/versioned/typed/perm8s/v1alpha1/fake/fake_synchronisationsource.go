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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"
	v1alpha1 "perm8s/pkg/apis/perm8s/v1alpha1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeSynchronisationSources implements SynchronisationSourceInterface
type FakeSynchronisationSources struct {
	Fake *FakePerm8sV1alpha1
	ns   string
}

var synchronisationsourcesResource = v1alpha1.SchemeGroupVersion.WithResource("synchronisationsources")

var synchronisationsourcesKind = v1alpha1.SchemeGroupVersion.WithKind("SynchronisationSource")

// Get takes name of the synchronisationSource, and returns the corresponding synchronisationSource object, and an error if there is any.
func (c *FakeSynchronisationSources) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.SynchronisationSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(synchronisationsourcesResource, c.ns, name), &v1alpha1.SynchronisationSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.SynchronisationSource), err
}

// List takes label and field selectors, and returns the list of SynchronisationSources that match those selectors.
func (c *FakeSynchronisationSources) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.SynchronisationSourceList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(synchronisationsourcesResource, synchronisationsourcesKind, c.ns, opts), &v1alpha1.SynchronisationSourceList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.SynchronisationSourceList{ListMeta: obj.(*v1alpha1.SynchronisationSourceList).ListMeta}
	for _, item := range obj.(*v1alpha1.SynchronisationSourceList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested synchronisationSources.
func (c *FakeSynchronisationSources) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(synchronisationsourcesResource, c.ns, opts))

}

// Create takes the representation of a synchronisationSource and creates it.  Returns the server's representation of the synchronisationSource, and an error, if there is any.
func (c *FakeSynchronisationSources) Create(ctx context.Context, synchronisationSource *v1alpha1.SynchronisationSource, opts v1.CreateOptions) (result *v1alpha1.SynchronisationSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(synchronisationsourcesResource, c.ns, synchronisationSource), &v1alpha1.SynchronisationSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.SynchronisationSource), err
}

// Update takes the representation of a synchronisationSource and updates it. Returns the server's representation of the synchronisationSource, and an error, if there is any.
func (c *FakeSynchronisationSources) Update(ctx context.Context, synchronisationSource *v1alpha1.SynchronisationSource, opts v1.UpdateOptions) (result *v1alpha1.SynchronisationSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(synchronisationsourcesResource, c.ns, synchronisationSource), &v1alpha1.SynchronisationSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.SynchronisationSource), err
}

// Delete takes name of the synchronisationSource and deletes it. Returns an error if one occurs.
func (c *FakeSynchronisationSources) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(synchronisationsourcesResource, c.ns, name, opts), &v1alpha1.SynchronisationSource{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeSynchronisationSources) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(synchronisationsourcesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.SynchronisationSourceList{})
	return err
}

// Patch applies the patch and returns the patched synchronisationSource.
func (c *FakeSynchronisationSources) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.SynchronisationSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(synchronisationsourcesResource, c.ns, name, pt, data, subresources...), &v1alpha1.SynchronisationSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.SynchronisationSource), err
}
