/*
Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file

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

package fake

import (
	"context"

	v1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeClusters implements ClusterInterface
type FakeClusters struct {
	Fake *FakeExtensionsV1alpha1
}

var clustersResource = schema.GroupVersionResource{Group: "extensions.gardener.cloud", Version: "v1alpha1", Resource: "clusters"}

var clustersKind = schema.GroupVersionKind{Group: "extensions.gardener.cloud", Version: "v1alpha1", Kind: "Cluster"}

// Get takes name of the cluster, and returns the corresponding cluster object, and an error if there is any.
func (c *FakeClusters) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.Cluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(clustersResource, name), &v1alpha1.Cluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Cluster), err
}

// List takes label and field selectors, and returns the list of Clusters that match those selectors.
func (c *FakeClusters) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.ClusterList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(clustersResource, clustersKind, opts), &v1alpha1.ClusterList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.ClusterList{ListMeta: obj.(*v1alpha1.ClusterList).ListMeta}
	for _, item := range obj.(*v1alpha1.ClusterList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested clusters.
func (c *FakeClusters) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(clustersResource, opts))
}

// Create takes the representation of a cluster and creates it.  Returns the server's representation of the cluster, and an error, if there is any.
func (c *FakeClusters) Create(ctx context.Context, cluster *v1alpha1.Cluster) (result *v1alpha1.Cluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(clustersResource, cluster), &v1alpha1.Cluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Cluster), err
}

// Update takes the representation of a cluster and updates it. Returns the server's representation of the cluster, and an error, if there is any.
func (c *FakeClusters) Update(ctx context.Context, cluster *v1alpha1.Cluster) (result *v1alpha1.Cluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(clustersResource, cluster), &v1alpha1.Cluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Cluster), err
}

// Delete takes name of the cluster and deletes it. Returns an error if one occurs.
func (c *FakeClusters) Delete(ctx context.Context, name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(clustersResource, name), &v1alpha1.Cluster{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeClusters) DeleteCollection(ctx context.Context, options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(clustersResource, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.ClusterList{})
	return err
}

// Patch applies the patch and returns the patched cluster.
func (c *FakeClusters) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Cluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(clustersResource, name, pt, data, subresources...), &v1alpha1.Cluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Cluster), err
}
