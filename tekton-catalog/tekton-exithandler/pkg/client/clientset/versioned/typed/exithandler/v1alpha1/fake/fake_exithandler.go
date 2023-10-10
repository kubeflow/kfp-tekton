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

	v1alpha1 "github.com/kubeflow/kfp-tekton/tekton-catalog/tekton-exithandler/pkg/apis/exithandler/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeExitHandlers implements ExitHandlerInterface
type FakeExitHandlers struct {
	Fake *FakeCustomV1alpha1
	ns   string
}

var exithandlersResource = schema.GroupVersionResource{Group: "custom.tekton.dev", Version: "v1alpha1", Resource: "exithandlers"}

var exithandlersKind = schema.GroupVersionKind{Group: "custom.tekton.dev", Version: "v1alpha1", Kind: "ExitHandler"}

// Get takes name of the exitHandler, and returns the corresponding exitHandler object, and an error if there is any.
func (c *FakeExitHandlers) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.ExitHandler, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(exithandlersResource, c.ns, name), &v1alpha1.ExitHandler{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ExitHandler), err
}

// List takes label and field selectors, and returns the list of ExitHandlers that match those selectors.
func (c *FakeExitHandlers) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.ExitHandlerList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(exithandlersResource, exithandlersKind, c.ns, opts), &v1alpha1.ExitHandlerList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.ExitHandlerList{ListMeta: obj.(*v1alpha1.ExitHandlerList).ListMeta}
	for _, item := range obj.(*v1alpha1.ExitHandlerList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested exitHandlers.
func (c *FakeExitHandlers) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(exithandlersResource, c.ns, opts))

}

// Create takes the representation of a exitHandler and creates it.  Returns the server's representation of the exitHandler, and an error, if there is any.
func (c *FakeExitHandlers) Create(ctx context.Context, exitHandler *v1alpha1.ExitHandler, opts v1.CreateOptions) (result *v1alpha1.ExitHandler, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(exithandlersResource, c.ns, exitHandler), &v1alpha1.ExitHandler{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ExitHandler), err
}

// Update takes the representation of a exitHandler and updates it. Returns the server's representation of the exitHandler, and an error, if there is any.
func (c *FakeExitHandlers) Update(ctx context.Context, exitHandler *v1alpha1.ExitHandler, opts v1.UpdateOptions) (result *v1alpha1.ExitHandler, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(exithandlersResource, c.ns, exitHandler), &v1alpha1.ExitHandler{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ExitHandler), err
}

// Delete takes name of the exitHandler and deletes it. Returns an error if one occurs.
func (c *FakeExitHandlers) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(exithandlersResource, c.ns, name, opts), &v1alpha1.ExitHandler{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeExitHandlers) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(exithandlersResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.ExitHandlerList{})
	return err
}

// Patch applies the patch and returns the patched exitHandler.
func (c *FakeExitHandlers) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ExitHandler, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(exithandlersResource, c.ns, name, pt, data, subresources...), &v1alpha1.ExitHandler{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ExitHandler), err
}
