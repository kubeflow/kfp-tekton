/*
Copyright 2020 The Knative Authors

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

package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"strconv"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"google.golang.org/protobuf/types/known/structpb"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/logging"

	"github.com/golang/protobuf/jsonpb"
)

type driverOptions struct {
	driverType  string
	options     driver.Options
	mlmdClient  *metadata.Client
	cacheClient *cacheutils.Client
}

const (
	// ReasonFailedValidation indicates that the reason for failure status is that Run failed runtime validation
	ReasonFailedValidation = "RunValidationFailed"

	// ReasonDriverError indicates that an error is throw while running the driver
	ReasonDriverError = "DriverError"

	ExecutionID    = "execution-id"
	Condition      = "condition"
	IterationCount = "iteration-count"
)

func DAGPublisher(ctx context.Context, opts driver.Options, mlmd *metadata.Client, status pb.Execution_State) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to publish driver DAG execution %s: %w", fmt.Sprint(opts.DAGExecutionID), err)
		}
	}()
	var outputParameters map[string]*structpb.Value
	execution, err := mlmd.GetExecution(ctx, opts.DAGExecutionID)
	if err != nil {
		return fmt.Errorf("failed to get execution: %w", err)
	}
	if err = mlmd.PublishExecution(ctx, execution, outputParameters, nil, status); err != nil {
		return fmt.Errorf("failed to publish: %w", err)
	}
	return nil
}

func UpdateDAGPublisher(ctx context.Context, options *driverOptions, status pb.Execution_State) (err error) {
	return DAGPublisher(ctx, options.options, options.mlmdClient, status)
}

func UpdateOptionsDAGExecutionID(options *driverOptions, DAGExecutionID string) {
	options.options.DAGExecutionID, _ = strconv.ParseInt(DAGExecutionID, 10, 64)
}

func UpdateOptionsIterationIndex(options *driverOptions, iterationIndex int) {
	options.options.IterationIndex = iterationIndex
}

func ExecDriver(ctx context.Context, options *driverOptions) (*[]tektonv1beta1.CustomRunResult, bool, string, string, string, error) {
	var execution *driver.Execution
	var err error
	logger := logging.FromContext(ctx)
	taskRunDecision := false
	executionID := ""
	executorInput := ""
	podSpecPatch := ""

	switch options.driverType {
	case "ROOT_DAG":
		execution, err = driver.RootDAG(ctx, options.options, options.mlmdClient)
	case "CONTAINER":
		execution, err = driver.Container(ctx, options.options, options.mlmdClient, options.cacheClient)
	case "DAG":
		execution, err = driver.DAG(ctx, options.options, options.mlmdClient)
	case "DAG_PUB":
		// current DAG_PUB only scheduled when the dag execution is completed
		err = DAGPublisher(ctx, options.options, options.mlmdClient, pb.Execution_COMPLETE)
		return &[]tektonv1beta1.CustomRunResult{}, taskRunDecision, executionID, executorInput, podSpecPatch, err
	default:
		err = fmt.Errorf("unknown driverType %s", options.driverType)
	}

	if err != nil {
		return nil, taskRunDecision, executionID, executorInput, podSpecPatch, err
	}

	var runResults []tektonv1beta1.CustomRunResult

	if execution.ID != 0 {
		logger.Infof("output execution.ID=%v", execution.ID)
		runResults = append(runResults, tektonv1beta1.CustomRunResult{
			Name:  ExecutionID,
			Value: fmt.Sprint(execution.ID),
		})
		executionID = fmt.Sprint(execution.ID)
	}
	if execution.IterationCount != nil {
		count := *execution.IterationCount
		// the count would be use as 'to' in PipelineLoop. since PipelineLoop's numberic iteration includes to,
		// need to substract 1 to compensate that.
		count = count - 1
		if count < 0 {
			count = 0
		}
		logger.Infof("output execution.IterationCount=%v, count:=%v", *execution.IterationCount, count)
		runResults = append(runResults, tektonv1beta1.CustomRunResult{
			Name:  IterationCount,
			Value: fmt.Sprint(count),
		})
	}

	logger.Infof("output execution.Condition=%v", execution.Condition)
	if execution.Condition == nil {
		runResults = append(runResults, tektonv1beta1.CustomRunResult{
			Name:  Condition,
			Value: "",
		})
	} else {
		runResults = append(runResults, tektonv1beta1.CustomRunResult{
			Name:  Condition,
			Value: strconv.FormatBool(*execution.Condition),
		})
	}

	if execution.ExecutorInput != nil {
		marshaler := jsonpb.Marshaler{}
		executorInputJSON, err := marshaler.MarshalToString(execution.ExecutorInput)
		if err != nil {
			return nil, taskRunDecision, executionID, executorInput, podSpecPatch, fmt.Errorf("failed to marshal ExecutorInput to JSON: %w", err)
		}
		logger.Infof("output ExecutorInput:%s\n", prettyPrint(executorInputJSON))
		executorInput = fmt.Sprint(executorInputJSON)
	}
	// seems no need to handle the PodSpecPatch
	if execution.Cached != nil {
		taskRunDecision = strconv.FormatBool(*execution.Cached) == "false"
	}

	if options.driverType == "CONTAINER" {
		podSpecPatch = execution.PodSpecPatch
	}

	return &runResults, taskRunDecision, executionID, executorInput, podSpecPatch, nil
}

func ParseParams(run *tektonv1beta1.CustomRun) (*driverOptions, *apis.FieldError) {
	if len(run.Spec.Params) == 0 {
		return nil, apis.ErrMissingField("params")
	}

	opts := &driverOptions{
		driverType: "",
	}
	opts.options.Namespace = run.Namespace
	mlmdOpts := metadata.ServerConfig{
		Address: "metadata-grpc-service.kubeflow.svc.cluster.local",
		Port:    "8080",
	}

	for _, param := range run.Spec.Params {
		switch param.Name {
		case "type":
			opts.driverType = param.Value.StringVal
		case "pipeline-name":
			opts.options.PipelineName = param.Value.StringVal
		case "run-id":
			opts.options.RunID = param.Value.StringVal
		case "component":
			if param.Value.StringVal != "" {
				componentSpec := &pipelinespec.ComponentSpec{}
				if err := jsonpb.UnmarshalString(param.Value.StringVal, componentSpec); err != nil {
					return nil, apis.ErrInvalidValue(
						fmt.Sprintf("failed to unmarshal component spec, error: %v\ncomponentSpec: %v", err, param.Value.StringVal),
						"component",
					)
				}
				opts.options.Component = componentSpec
			}
		case "task":
			if param.Value.StringVal != "" {
				taskSpec := &pipelinespec.PipelineTaskSpec{}
				if err := jsonpb.UnmarshalString(param.Value.StringVal, taskSpec); err != nil {
					return nil, apis.ErrInvalidValue(
						fmt.Sprintf("failed to unmarshal task spec, error: %v\ntask: %v", err, param.Value.StringVal),
						"task",
					)
				}
				opts.options.Task = taskSpec
			}
		case "runtime-config":
			if param.Value.StringVal != "" {
				runtimeConfig := &pipelinespec.PipelineJob_RuntimeConfig{}
				if err := jsonpb.UnmarshalString(param.Value.StringVal, runtimeConfig); err != nil {
					return nil, apis.ErrInvalidValue(
						fmt.Sprintf("failed to unmarshal runtime config, error: %v\nruntimeConfig: %v", err, param.Value.StringVal),
						"runtime-config",
					)
				}
				opts.options.RuntimeConfig = runtimeConfig
			}
		case "container":
			if param.Value.StringVal != "" {
				containerSpec := &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{}
				if err := jsonpb.UnmarshalString(param.Value.StringVal, containerSpec); err != nil {
					return nil, apis.ErrInvalidValue(
						fmt.Sprintf("failed to unmarshal container spec, error: %v\ncontainerSpec: %v", err, param.Value.StringVal),
						"container",
					)
				}
				opts.options.Container = containerSpec
			}
		case "iteration-index":
			if param.Value.StringVal != "" {
				tmp, _ := strconv.ParseInt(param.Value.StringVal, 10, 32)
				opts.options.IterationIndex = int(tmp)
			} else {
				opts.options.IterationIndex = -1
			}
		case "dag-execution-id":
			if param.Value.StringVal != "" {
				opts.options.DAGExecutionID, _ = strconv.ParseInt(param.Value.StringVal, 10, 64)
			}
		case "mlmd-server-address":
			mlmdOpts.Address = param.Value.StringVal
		case "mlmd-server-port":
			mlmdOpts.Port = param.Value.StringVal
		case "kubernetes-config":
			var k8sExecCfg *kubernetesplatform.KubernetesExecutorConfig
			if param.Value.StringVal != "" {
				k8sExecCfg = &kubernetesplatform.KubernetesExecutorConfig{}
				if err := jsonpb.UnmarshalString(param.Value.StringVal, k8sExecCfg); err != nil {
					return nil, apis.ErrInvalidValue(
						fmt.Sprintf("failed to unmarshal Kubernetes config, error: %v\nKubernetesConfig: %v", err, param.Value.StringVal),
						"kubernetes-config",
					)
				}
				opts.options.KubernetesExecutorConfig = k8sExecCfg
			}
		}
	}

	//Check all options
	if opts.driverType == "" {
		return nil, apis.ErrMissingField("type")
	}

	if opts.options.RunID == "" {
		return nil, apis.ErrMissingField("run-id")
	}

	if opts.driverType == "ROOT_DAG" && opts.options.RuntimeConfig == nil {
		return nil, apis.ErrMissingField("runtime-config")
	}

	client, err := metadata.NewClient(mlmdOpts.Address, mlmdOpts.Port)
	if err != nil {
		return nil, apis.ErrGeneric(fmt.Sprintf("can't estibalish MLMD connection: %v", err))
	}
	opts.mlmdClient = client
	cacheClient, err := cacheutils.NewClient()
	if err != nil {
		return nil, apis.ErrGeneric(fmt.Sprintf("can't estibalish cache connection: %v", err))
	}
	opts.cacheClient = cacheClient
	return opts, nil
}

func prettyPrint(jsonStr string) string {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, []byte(jsonStr), "", "  ")
	if err != nil {
		return jsonStr
	}
	return prettyJSON.String()
}
