// Copyright 2018 Google LLC
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
	"time"

	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/typed/pipeline/v1beta1"
	"k8s.io/client-go/rest"
)

type TektonClientInterface interface {
	PipelineRun(namespace string) tektonv1beta1.PipelineRunInterface
}

type TektonClient struct {
	tektonClient tektonv1beta1.TektonV1beta1Interface
}

func (tektonClient *TektonClient) PipelineRun(namespace string) tektonv1beta1.PipelineRunInterface {
	return tektonClient.tektonClient.PipelineRuns(namespace)
}

func NewTektonClientOrFatal(initConnectionTimeout time.Duration, clientParams util.ClientParameters) *TektonClient {
	var tektonClient tektonv1beta1.TektonV1beta1Interface
	var operation = func() error {
		restConfig, err := rest.InClusterConfig()
		if err != nil {
			return errors.Wrap(err, "Failed to initialize the RestConfig")
		}
		restConfig.QPS = float32(clientParams.QPS)
		restConfig.Burst = clientParams.Burst
		tektonClient = tektonclient.NewForConfigOrDie(restConfig).TektonV1beta1()
		return nil
	}

	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initConnectionTimeout
	err := backoff.Retry(operation, b)

	if err != nil {
		glog.Fatalf("Failed to create TektonClient. Error: %v", err)
	}
	return &TektonClient{tektonClient}
}
