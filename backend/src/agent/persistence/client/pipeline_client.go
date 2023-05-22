// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"context"
	"fmt"
	"time"

	api "github.com/kubeflow/pipelines/backend/api/v1/go_client"
	kfpResource "github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	kfpServer "github.com/kubeflow/pipelines/backend/src/apiserver/server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
)

const (
	addressTemp = "%s:%s"
)

type PipelineClientInterface interface {
	ReportWorkflow(workflow *util.Workflow) error
	ReportScheduledWorkflow(swf *util.ScheduledWorkflow) error
	ReadArtifact(request *api.ReadArtifactRequest) (*api.ReadArtifactResponse, error)
	ReportRunMetrics(request *api.ReportRunMetricsRequest) (*api.ReportRunMetricsResponse, error)
}

type PipelineClient struct {
	initializeTimeout   time.Duration
	timeout             time.Duration
	reportServiceClient api.ReportServiceClient
	runServiceClient    api.RunServiceClient
	resourceManager     *kfpResource.ResourceManager
}

func NewPipelineClient(
	initializeTimeout time.Duration,
	timeout time.Duration,
	basePath string,
	mlPipelineServiceName string,
	mlPipelineServiceHttpPort string,
	mlPipelineServiceGRPCPort string) (*PipelineClient, error) {
	httpAddress := fmt.Sprintf(addressTemp, mlPipelineServiceName, mlPipelineServiceHttpPort)
	grpcAddress := fmt.Sprintf(addressTemp, mlPipelineServiceName, mlPipelineServiceGRPCPort)
	clientManager := newClientManager()
	resourceManager := kfpResource.NewResourceManager(&clientManager)
	err := util.WaitForAPIAvailable(initializeTimeout, basePath, httpAddress)
	if err != nil {
		return nil, errors.Wrapf(err,
			"Failed to initialize pipeline client. Error: %s", err.Error())
	}
	connection, err := util.GetRpcConnection(grpcAddress)
	if err != nil {
		return nil, errors.Wrapf(err,
			"Failed to get RPC connection. Error: %s", err.Error())
	}

	return &PipelineClient{
		initializeTimeout:   initializeTimeout,
		timeout:             timeout,
		reportServiceClient: api.NewReportServiceClient(connection),
		runServiceClient:    api.NewRunServiceClient(connection),
		resourceManager:     resourceManager,
	}, nil
}

func (p *PipelineClient) ReportWorkflow(workflow *util.Workflow) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	err := p.resourceManager.ReportWorkflowResource(ctx, workflow)

	if err != nil {
		return util.Wrap(err, "Report workflow failed.")
	}

	return nil
}

func (p *PipelineClient) ReportScheduledWorkflow(swf *util.ScheduledWorkflow) error {
	_, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	err := p.resourceManager.ReportScheduledWorkflowResource(swf)

	if err != nil {
		return util.Wrap(err, "Report scheduled workflow failed.")
	}

	return nil
}

// ReadArtifact reads artifact content from run service. If the artifact is not present, returns
// nil response.
func (p *PipelineClient) ReadArtifact(request *api.ReadArtifactRequest) (*api.ReadArtifactResponse, error) {
	_, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	content, err := p.resourceManager.ReadArtifact(request.GetRunId(), request.GetNodeId(), request.GetArtifactName())
	if err != nil {
		// TODO: check NotFound error code before skip the error.
		return nil, nil
	}
	response := &api.ReadArtifactResponse{
		Data: content,
	}

	return response, nil
}

// ReportRunMetrics reports run metrics to run service.
func (p *PipelineClient) ReportRunMetrics(request *api.ReportRunMetricsRequest) (*api.ReportRunMetricsResponse, error) {
	_, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// Makes sure run exists
	_, err := p.resourceManager.GetRun(request.GetRunId())
	if err != nil {
		return nil, err
	}
	response := &api.ReportRunMetricsResponse{
		Results: []*api.ReportRunMetricsResponse_ReportRunMetricResult{},
	}
	for _, metric := range request.GetMetrics() {
		err := kfpServer.ValidateRunMetric(metric)
		if err == nil {
			err = p.resourceManager.ReportMetric(metric, request.GetRunId())
		}
		response.Results = append(
			response.Results,
			kfpServer.NewReportRunMetricResult(metric.GetName(), metric.GetNodeId(), err))
	}
	return response, nil
}
