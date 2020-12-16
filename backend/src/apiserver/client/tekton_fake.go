// Copyright 2019 Google LLC
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
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/typed/pipeline/v1beta1"
)

type FakeTektonClient struct {
	workflowClientFake *FakeWorkflowClient
}

func NewFakeTektonClient() *FakeTektonClient {
	return &FakeTektonClient{NewWorkflowClientFake()}
}

func (c *FakeTektonClient) Workflow(namespace string) tektonv1beta1.PipelineRunInterface {
	if len(namespace) == 0 {
		panic(util.NewResourceNotFoundError("Namespace", namespace))
	}
	return c.workflowClientFake
}

func (c *FakeTektonClient) PipelineRun(namespace string) tektonv1beta1.PipelineRunInterface {
	if len(namespace) == 0 {
		panic(util.NewResourceNotFoundError("Namespace", namespace))
	}
	return c.workflowClientFake
}

func (c *FakeTektonClient) GetWorkflowCount() int {
	return len(c.workflowClientFake.workflows)
}

func (c *FakeTektonClient) GetWorkflowKeys() map[string]bool {
	result := map[string]bool{}
	for key := range c.workflowClientFake.workflows {
		result[key] = true
	}
	return result
}

func (c *FakeTektonClient) IsTerminated(name string) (bool, error) {
	_, ok := c.workflowClientFake.workflows[name]
	if !ok {
		return false, errors.New("No workflow found with name: " + name)
	}
	// There's no activeDeadlineSeconds in Tekton

	return true, nil
}

type FakeTektonClientWithBadWorkflow struct {
	workflowClientFake *FakeBadWorkflowClient
}

func NewFakeTektonClientWithBadWorkflow() *FakeTektonClientWithBadWorkflow {
	return &FakeTektonClientWithBadWorkflow{&FakeBadWorkflowClient{}}
}

func (c *FakeTektonClientWithBadWorkflow) Workflow(namespace string) tektonv1beta1.PipelineRunInterface {
	return c.workflowClientFake
}
