package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/golang/protobuf/jsonpb"
	v1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1alpha1/run"
	v1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/logging"
	reconciler "knative.dev/pkg/reconciler"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/v2/driver"
	"github.com/kubeflow/pipelines/v2/metadata"
)

const (
	// ReasonFailedValidation indicates that the reason for failure status is that Run failed runtime validation
	ReasonFailedValidation = "RunValidationFailed"

	// ReasonDriverError indicates that an error is throw while running the driver
	ReasonDriverError = "DriverError"

	// ReasonDriverSuccess indicates that driver finished successfully
	ReasonDriverSuccess = "DriverSuccess"
)

// newReconciledNormal makes a new reconciler event with event type Normal, and reason RunReconciled.
func newReconciledNormal(namespace, name string) reconciler.Event {
	return reconciler.NewEvent(v1.EventTypeNormal, "RunReconciled", "Run reconciled: \"%s/%s\"", namespace, name)
}

// Reconciler implements controller.Reconciler for Run resources.
type Reconciler struct {
}

type outputParams struct {
	executionID   string
	contextID     string
	executorInput string
}

type driverOptions struct {
	driverType string
	options    driver.Options
	mlmdClient *metadata.Client
	outputs    outputParams
}

// Check that our Reconciler implements Interface
var _ run.Interface = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, run *v1alpha1.Run) reconciler.Event {
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
		run.Status.MarkRunFailed(ReasonFailedValidation,
			"Run can't be run because it has an invalid param - %v", err)
		return nil
	}

	runResults, driverErr := execDriver(ctx, options)
	if driverErr != nil {
		logger.Errorf("kfp-driver execution failed when reconciling Run %s/%s: %v", run.Namespace, run.Name, driverErr)
		run.Status.MarkRunFailed(ReasonDriverError,
			"kfp-driver execution failed: %v", driverErr)
		return nil
	}

	run.Status.Results = append(run.Status.Results, *runResults...)
	run.Status.MarkRunSucceeded(ReasonDriverSuccess, "kfp-driver finished successfully")
	return newReconciledNormal(run.Namespace, run.Name)
}

func execDriver(ctx context.Context, options *driverOptions) (*[]v1alpha1.RunResult, error) {
	var execution *driver.Execution
	var err error
	logger := logging.FromContext(ctx)

	switch options.driverType {
	case "ROOT_DAG":
		execution, err = driver.RootDAG(ctx, options.options, options.mlmdClient)
	case "CONTAINER":
		execution, err = driver.Container(ctx, options.options, options.mlmdClient)
	default:
		err = fmt.Errorf("unknown driverType %s", options.driverType)
	}

	if err != nil {
		return nil, err
	}

	var runResults []v1alpha1.RunResult

	if execution.ID != 0 {
		logger.Infof("output execution.ID=%v", execution.ID)
		runResults = append(runResults, v1alpha1.RunResult{
			Name:  options.outputs.executionID,
			Value: fmt.Sprint(execution.ID),
		})
	}
	if execution.Context != 0 {
		logger.Infof("output execution.Context=%v", execution.Context)
		runResults = append(runResults, v1alpha1.RunResult{
			Name:  options.outputs.contextID,
			Value: fmt.Sprint(execution.Context),
		})
	}
	if execution.ExecutorInput != nil {
		marshaler := jsonpb.Marshaler{}
		executorInputJSON, err := marshaler.MarshalToString(execution.ExecutorInput)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal ExecutorInput to JSON: %w", err)
		}
		logger.Infof("output ExecutorInput:%s\n", prettyPrint(executorInputJSON))
		runResults = append(runResults, v1alpha1.RunResult{
			Name:  options.outputs.executorInput,
			Value: fmt.Sprint(executorInputJSON),
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
	return string(prettyJSON.Bytes())
}

func parseParams(run *v1alpha1.Run) (*driverOptions, *apis.FieldError) {
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
			componentSpec := &pipelinespec.ComponentSpec{}
			if err := jsonpb.UnmarshalString(param.Value.StringVal, componentSpec); err != nil {
				return nil, apis.ErrInvalidValue(
					fmt.Sprintf("failed to unmarshal component spec, error: %v\ncomponentSpec: %v", err, param.Value.StringVal),
					"component",
				)
			}
			opts.options.Component = componentSpec
		case "task":
			taskSpec := &pipelinespec.PipelineTaskSpec{}
			if err := jsonpb.UnmarshalString(param.Value.StringVal, taskSpec); err != nil {
				return nil, apis.ErrInvalidValue(
					fmt.Sprintf("failed to unmarshal task spec, error: %v\ntask: %v", err, param.Value.StringVal),
					"task",
				)
			}
			opts.options.Task = taskSpec
		case "runtime-config":
			runtimeConfig := &pipelinespec.PipelineJob_RuntimeConfig{}
			if err := jsonpb.UnmarshalString(param.Value.StringVal, runtimeConfig); err != nil {
				return nil, apis.ErrInvalidValue(
					fmt.Sprintf("failed to unmarshal runtime config, error: %v\nruntimeConfig: %v", err, param.Value.StringVal),
					"runtime-config",
				)
			}
			opts.options.RuntimeConfig = runtimeConfig
		case "dag-context-id":
			opts.options.DAGContextID, _ = strconv.ParseInt(param.Value.StringVal, 10, 64)
		case "dag-execution-id":
			opts.options.DAGExecutionID, _ = strconv.ParseInt(param.Value.StringVal, 10, 64)
		case "mlmd-server-address":
			mlmdOpts.Address = param.Value.StringVal
		case "mlmd-server-port":
			mlmdOpts.Port = param.Value.StringVal
		case "execution-id":
			opts.outputs.executionID = param.Value.StringVal
		case "context-id":
			opts.outputs.contextID = param.Value.StringVal
		case "executor-input":
			opts.outputs.executorInput = param.Value.StringVal
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
	return opts, nil
}
