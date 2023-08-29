// Copyright 2018 The Kubeflow Authors
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

package client

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/kubeflow/pipelines/backend/src/common/util"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	tektonV1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8schema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

type FakeWorkflowClient struct {
	workflows       map[string]*tektonV1.PipelineRun
	lastGeneratedId int
}

func NewWorkflowClientFake() *FakeWorkflowClient {
	return &FakeWorkflowClient{
		workflows:       make(map[string]*tektonV1.PipelineRun),
		lastGeneratedId: -1,
	}
}

func (c *FakeWorkflowClient) Create(ctx context.Context, workflow *tektonV1.PipelineRun, options v1.CreateOptions) (*tektonV1.PipelineRun, error) {
	if workflow.GenerateName != "" {
		c.lastGeneratedId += 1
		workflow.Name = workflow.GenerateName + strconv.Itoa(c.lastGeneratedId)
		workflow.GenerateName = ""
	}
	c.workflows[workflow.Name] = workflow
	return workflow, nil
}

func (c *FakeWorkflowClient) Get(ctx context.Context, name string, options v1.GetOptions) (*tektonV1.PipelineRun, error) {
	workflow, ok := c.workflows[name]
	if ok {
		return workflow, nil
	}
	return nil, k8errors.NewNotFound(k8schema.ParseGroupResource("tekton.dev"), name)
}

func (c *FakeWorkflowClient) UpdateStatus(ctx context.Context, workflow *tektonV1.PipelineRun, options v1.UpdateOptions) (*tektonV1.PipelineRun, error) {
	return workflow, nil
}

func (c *FakeWorkflowClient) List(ctx context.Context, opts v1.ListOptions) (*tektonV1.PipelineRunList, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (c *FakeWorkflowClient) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (c *FakeWorkflowClient) Update(ctx context.Context, workflow *tektonV1.PipelineRun, options v1.UpdateOptions) (*tektonV1.PipelineRun, error) {
	name := workflow.GetObjectMeta().GetName()
	_, ok := c.workflows[name]
	if ok {
		return workflow, nil
	}
	return nil, k8errors.NewNotFound(k8schema.ParseGroupResource("tekton.dev"), name)
}

func (c *FakeWorkflowClient) Delete(ctx context.Context, name string, options v1.DeleteOptions) error {
	_, ok := c.workflows[name]
	if ok {
		return nil
	}
	return k8errors.NewNotFound(k8schema.ParseGroupResource("tekton.dev"), name)
}

func (c *FakeWorkflowClient) DeleteCollection(ctx context.Context, options v1.DeleteOptions,
	listOptions v1.ListOptions) error {
	glog.Error("This fake method is not yet implemented.")
	return nil
}

func (c *FakeWorkflowClient) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, options v1.PatchOptions,
	subresources ...string) (*tektonV1.PipelineRun, error) {

	_, ok := c.workflows[name]
	if !ok {
		return nil, k8errors.NewNotFound(k8schema.ParseGroupResource("tekton.dev"), name)
	}

	var dat map[string]interface{}
	json.Unmarshal(data, &dat)

	// TODO: Should we actually assert the type here, or just panic if it's wrong?

	if _, ok := dat["spec"]; ok {
		spec := dat["spec"].(map[string]interface{})
		// There's no activeDeadlineSeconds in Tekton
		activeDeadlineSeconds := spec["activeDeadlineSeconds"].(float64)

		// Simulate terminating a workflow
		if pt == types.MergePatchType && activeDeadlineSeconds == 0 {
			workflow, ok := c.workflows[name]
			if ok {
				return workflow, nil
			}
		}
	}

	if _, ok := dat["metadata"]; ok {
		workflow, ok := c.workflows[name]
		if ok {
			if workflow.Labels == nil {
				workflow.Labels = map[string]string{}
			}
			workflow.Labels[util.LabelKeyWorkflowPersistedFinalState] = "true"
			return workflow, nil
		}
	}
	return nil, errors.New("Failed to patch workflow")
}

type FakeBadWorkflowClient struct {
	FakeWorkflowClient
}

func (FakeBadWorkflowClient) Create(ctx context.Context, workflow *tektonV1.PipelineRun, options v1.CreateOptions) (*tektonV1.PipelineRun, error) {
	return nil, errors.New("some error")
}

func (FakeBadWorkflowClient) Get(ctx context.Context, name string, options v1.GetOptions) (*tektonV1.PipelineRun, error) {
	return nil, errors.New("some error")
}

func (c *FakeBadWorkflowClient) Update(ctx context.Context, workflow *tektonV1.PipelineRun, options v1.UpdateOptions) (*tektonV1.PipelineRun, error) {
	return nil, errors.New("failed to update workflow")
}

func (c *FakeBadWorkflowClient) Delete(ctx context.Context, name string, options v1.DeleteOptions) error {
	return errors.New("failed to delete workflow")
}
