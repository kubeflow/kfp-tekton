// Copyright 2023 kubeflow.org
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/kubeflow/kfp-tekton/tekton-catalog/tekton-kfptask/pkg/apis/kfptask/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeKfpTasks implements KfpTaskInterface
type FakeKfpTasks struct {
	Fake *FakeCustomV1alpha1
	ns   string
}

var kfptasksResource = schema.GroupVersionResource{Group: "custom.tekton.dev", Version: "v1alpha1", Resource: "kfptasks"}

var kfptasksKind = schema.GroupVersionKind{Group: "custom.tekton.dev", Version: "v1alpha1", Kind: "KfpTask"}

// Get takes name of the kfpTask, and returns the corresponding kfpTask object, and an error if there is any.
func (c *FakeKfpTasks) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.KfpTask, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(kfptasksResource, c.ns, name), &v1alpha1.KfpTask{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.KfpTask), err
}

// List takes label and field selectors, and returns the list of KfpTasks that match those selectors.
func (c *FakeKfpTasks) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.KfpTaskList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(kfptasksResource, kfptasksKind, c.ns, opts), &v1alpha1.KfpTaskList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.KfpTaskList{ListMeta: obj.(*v1alpha1.KfpTaskList).ListMeta}
	for _, item := range obj.(*v1alpha1.KfpTaskList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested kfpTasks.
func (c *FakeKfpTasks) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(kfptasksResource, c.ns, opts))

}

// Create takes the representation of a kfpTask and creates it.  Returns the server's representation of the kfpTask, and an error, if there is any.
func (c *FakeKfpTasks) Create(ctx context.Context, kfpTask *v1alpha1.KfpTask, opts v1.CreateOptions) (result *v1alpha1.KfpTask, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(kfptasksResource, c.ns, kfpTask), &v1alpha1.KfpTask{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.KfpTask), err
}

// Update takes the representation of a kfpTask and updates it. Returns the server's representation of the kfpTask, and an error, if there is any.
func (c *FakeKfpTasks) Update(ctx context.Context, kfpTask *v1alpha1.KfpTask, opts v1.UpdateOptions) (result *v1alpha1.KfpTask, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(kfptasksResource, c.ns, kfpTask), &v1alpha1.KfpTask{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.KfpTask), err
}

// Delete takes name of the kfpTask and deletes it. Returns an error if one occurs.
func (c *FakeKfpTasks) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(kfptasksResource, c.ns, name, opts), &v1alpha1.KfpTask{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeKfpTasks) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(kfptasksResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.KfpTaskList{})
	return err
}

// Patch applies the patch and returns the patched kfpTask.
func (c *FakeKfpTasks) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.KfpTask, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(kfptasksResource, c.ns, name, pt, data, subresources...), &v1alpha1.KfpTask{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.KfpTask), err
}
