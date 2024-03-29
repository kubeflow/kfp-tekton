// Copyright 2020 Google LLC
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
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FakeTektonClient struct {
}

func (c *FakeTektonClient) GetTaskRun(namespace, name string, queryOptions metav1.GetOptions) (*v1.TaskRun, error) {
	tr := &v1.TaskRun{
		Spec: v1.TaskRunSpec{
			Status: v1.TaskRunSpecStatusCancelled,
		},
		Status: v1.TaskRunStatus{},
	}

	return tr, nil
}

func NewFakeTektonClient() *FakeTektonClient {
	return &FakeTektonClient{}
}
