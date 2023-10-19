package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/golang/protobuf/jsonpb"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/customrun"
	v1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/logging"
	reconciler "knative.dev/pkg/reconciler"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
)

const (
	// ReasonFailedValidation indicates that the reason for failure status is that Run failed runtime validation
	ReasonFailedValidation = "RunValidationFailed"

	// ReasonDriverError indicates that an error is throw while running the driver
	ReasonDriverError = "DriverError"

	// ReasonDriverSuccess indicates that driver finished successfully
	ReasonDriverSuccess = "DriverSuccess"

	ExecutionID    = "execution-id"
	ExecutorInput  = "executor-input"
	CacheDecision  = "cached-decision"
	Condition      = "condition"
	IterationCount = "iteration-count"
	PodSpecPatch   = "pod-spec-patch"
)

// newReconciledNormal makes a new reconciler event with event type Normal, and reason RunReconciled.
func newReconciledNormal(namespace, name string) reconciler.Event {
	return reconciler.NewEvent(v1.EventTypeNormal, "RunReconciled", "Run reconciled: \"%s/%s\"", namespace, name)
}

// Reconciler implements controller.Reconciler for Run resources.
type Reconciler struct {
}

type driverOptions struct {
	driverType  string
	options     driver.Options
	mlmdClient  *metadata.Client
	cacheClient *cacheutils.Client
}

// Check that our Reconciler implements Interface
var _ customrun.Interface = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, run *v1beta1.CustomRun) reconciler.Event {
	logger := logging.FromContext(ctx)
	logger.Infof("Reconciling Run %s/%s", run.Namespace, run.Name)

	// If the Run has not started, initialize the Condition and set the start time.
	if !run.HasStarted() {
		logger.Infof("Starting new Run %s/%s", run.Namespace, run.Name)
		run.Status.InitializeConditions()
		// In case node time was not synchronized, when controller has been scheduled to other nodes.
		if run.Status.StartTime.Sub(run.CreationTimestamp.Time) < 0 {
			logger.Warnf("Run %s/%s createTimestamp %s is after the Run started %s", run.Namespace, run.Name, run.CreationTimestamp, run.Status.StartTime)
			run.Status.StartTime = &run.CreationTimestamp
		}
	}

	if run.IsDone() {
		logger.Infof("Run %s/%s is done", run.Namespace, run.Name)
		return nil
	}

	options, err := parseParams(run)
	if err != nil {
		logger.Errorf("Run %s/%s is invalid because of %s", run.Namespace, run.Name, err)
		run.Status.MarkCustomRunFailed(ReasonFailedValidation,
			"Run can't be run because it has an invalid param - %v", err)
		return nil
	}

	runResults, driverErr := execDriver(ctx, options)
	if driverErr != nil {
		logger.Errorf("kfp-driver execution failed when reconciling Run %s/%s: %v", run.Namespace, run.Name, driverErr)
		run.Status.MarkCustomRunFailed(ReasonDriverError,
			"kfp-driver execution failed: %v", driverErr)
		return nil
	}

	run.Status.Results = append(run.Status.Results, *runResults...)
	run.Status.MarkCustomRunSucceeded(ReasonDriverSuccess, "kfp-driver finished successfully")
	return newReconciledNormal(run.Namespace, run.Name)
}

func execDriver(ctx context.Context, options *driverOptions) (*[]v1beta1.CustomRunResult, error) {
	var execution *driver.Execution
	var err error
	logger := logging.FromContext(ctx)

	switch options.driverType {
	case "ROOT_DAG":
		execution, err = driver.RootDAG(ctx, options.options, options.mlmdClient)
	case "CONTAINER":
		execution, err = driver.Container(ctx, options.options, options.mlmdClient, options.cacheClient)
	case "DAG":
		execution, err = driver.DAG(ctx, options.options, options.mlmdClient)
	case "DAG-PUB":
		// no-op for now
		return &[]v1beta1.CustomRunResult{}, nil
	default:
		err = fmt.Errorf("unknown driverType %s", options.driverType)
	}

	if err != nil {
		return nil, err
	}

	var runResults []v1beta1.CustomRunResult

	if execution.ID != 0 {
		logger.Infof("output execution.ID=%v", execution.ID)
		runResults = append(runResults, v1beta1.CustomRunResult{
			Name:  ExecutionID,
			Value: fmt.Sprint(execution.ID),
		})
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
		runResults = append(runResults, v1beta1.CustomRunResult{
			Name:  IterationCount,
			Value: fmt.Sprint(count),
		})
	}

	logger.Infof("output execution.Condition=%v", execution.Condition)
	if execution.Condition == nil {
		runResults = append(runResults, v1beta1.CustomRunResult{
			Name:  Condition,
			Value: "",
		})
	} else {
		runResults = append(runResults, v1beta1.CustomRunResult{
			Name:  Condition,
			Value: strconv.FormatBool(*execution.Condition),
		})
	}

	if execution.ExecutorInput != nil {
		marshaler := jsonpb.Marshaler{}
		executorInputJSON, err := marshaler.MarshalToString(execution.ExecutorInput)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal ExecutorInput to JSON: %w", err)
		}
		logger.Infof("output ExecutorInput:%s\n", prettyPrint(executorInputJSON))
		runResults = append(runResults, v1beta1.CustomRunResult{
			Name:  ExecutorInput,
			Value: fmt.Sprint(executorInputJSON),
		})
	}
	// seems no need to handle the PodSpecPatch
	if execution.Cached != nil {
		runResults = append(runResults, v1beta1.CustomRunResult{
			Name:  CacheDecision,
			Value: strconv.FormatBool(*execution.Cached),
		})
	}

	if options.driverType == "CONTAINER" {
		runResults = append(runResults, v1beta1.CustomRunResult{
			Name:  PodSpecPatch,
			Value: execution.PodSpecPatch,
		})
	}

	return &runResults, nil
}

func prettyPrint(jsonStr string) string {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, []byte(jsonStr), "", "  ")
	if err != nil {
		return jsonStr
	}
	return prettyJSON.String()
}

func parseParams(run *v1beta1.CustomRun) (*driverOptions, *apis.FieldError) {
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
		case "pipeline_name":
			opts.options.PipelineName = param.Value.StringVal
		case "run_id":
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
		case "runtime_config":
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
		case "iteration_index":
			if param.Value.StringVal != "" {
				tmp, _ := strconv.ParseInt(param.Value.StringVal, 10, 32)
				opts.options.IterationIndex = int(tmp)
			} else {
				opts.options.IterationIndex = -1
			}
		case "dag_execution_id":
			if param.Value.StringVal != "" {
				opts.options.DAGExecutionID, _ = strconv.ParseInt(param.Value.StringVal, 10, 64)
			}
		case "mlmd_server_address":
			mlmdOpts.Address = param.Value.StringVal
		case "mlmd_server_port":
			mlmdOpts.Port = param.Value.StringVal
		case "kubernetes_config":
			var k8sExecCfg *kubernetesplatform.KubernetesExecutorConfig
			if param.Value.StringVal != "" {
				k8sExecCfg = &kubernetesplatform.KubernetesExecutorConfig{}
				if err := jsonpb.UnmarshalString(param.Value.StringVal, k8sExecCfg); err != nil {
					return nil, apis.ErrInvalidValue(
						fmt.Sprintf("failed to unmarshal Kubernetes config, error: %v\nKubernetesConfig: %v", err, param.Value.StringVal),
						"kubernetes_config",
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
