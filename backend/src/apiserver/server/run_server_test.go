package server

import (
	"context"
	"testing"

	api "github.com/kubeflow/pipelines/backend/api/v1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	authorizationv1 "k8s.io/api/authorization/v1"
)

// Converted argo v1alpha1.workflow to tekton v1beta1.pipelinerun
// Removed conflicted v1alpha1.parameters.

func TestCreateRun(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &api.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
		},
	}
	_, err := server.CreateRun(nil, &api.CreateRunRequest{Run: run})
	assert.Nil(t, err)

	expectedRuntimeWorkflow := testWorkflow.DeepCopy()
	expectedRuntimeWorkflow.Labels = map[string]string{util.LabelKeyWorkflowRunId: "123e4567-e89b-12d3-a456-426655440000"}
	expectedRuntimeWorkflow.Annotations = map[string]string{util.AnnotationKeyRunName: "run1"}
	expectedRuntimeWorkflow.Spec.ServiceAccountName = "pipeline-runner"
}

func TestCreateRunPatch(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &api.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflowPatch.ToStringForStore(),
		},
	}
	_, err := server.CreateRun(nil, &api.CreateRunRequest{Run: run})
	assert.Nil(t, err)

	expectedRuntimeWorkflow := testWorkflowPatch.DeepCopy()
	expectedRuntimeWorkflow.Labels = map[string]string{util.LabelKeyWorkflowRunId: "123e4567-e89b-12d3-a456-426655440000"}
	expectedRuntimeWorkflow.Annotations = map[string]string{util.AnnotationKeyRunName: "run1"}
	expectedRuntimeWorkflow.Spec.ServiceAccountName = "pipeline-runner"
}

func TestCreateRun_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &api.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	_, err := server.CreateRun(ctx, &api.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: "ns1",
		Verb:      common.RbacResourceVerbCreate,
		Group:     common.RbacPipelinesGroup,
		Version:   common.RbacPipelinesVersion,
		Resource:  common.RbacResourceTypeRuns,
		Name:      "run1",
	}
	assert.EqualError(
		t,
		err,
		wrapFailedAuthzRequestError(wrapFailedAuthzApiResourcesError(getPermissionDeniedError(userIdentity, resourceAttributes))).Error(),
	)
}

func TestCreateRun_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	viper.Set(common.DefaultPipelineRunnerServiceAccountFlag, "default-editor")
	defer viper.Set(common.MultiUserMode, "false")
	defer viper.Set(common.DefaultPipelineRunnerServiceAccountFlag, "pipeline-runner")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &api.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
		},
	}
	_, err := server.CreateRun(ctx, &api.CreateRunRequest{Run: run})
	assert.Nil(t, err)

	expectedRuntimeWorkflow := testWorkflow.DeepCopy()
	expectedRuntimeWorkflow.Labels = map[string]string{util.LabelKeyWorkflowRunId: "123e4567-e89b-12d3-a456-426655440000"}
	expectedRuntimeWorkflow.Annotations = map[string]string{util.AnnotationKeyRunName: "run1"}
	expectedRuntimeWorkflow.Spec.ServiceAccountName = "default-editor" // In multi-user mode, we use default service account.
}

func TestListRun(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &api.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
		},
	}
	_, err := server.CreateRun(nil, &api.CreateRunRequest{Run: run})
	assert.Nil(t, err)

}

func TestListRuns_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	_, err := server.ListRuns(ctx, &api.ListRunsRequest{
		ResourceReferenceKey: &api.ResourceKey{
			Type: api.ResourceType_NAMESPACE,
			Id:   "ns1",
		},
	})
	assert.NotNil(t, err)
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: "ns1",
		Verb:      common.RbacResourceVerbList,
		Group:     common.RbacPipelinesGroup,
		Version:   common.RbacPipelinesVersion,
		Resource:  common.RbacResourceTypeRuns,
	}
	assert.EqualError(
		t,
		err,
		util.Wrap(
			wrapFailedAuthzApiResourcesError(getPermissionDeniedError(userIdentity, resourceAttributes)),
			"Failed to authorize with namespace resource reference.").Error(),
	)
}

// Removed tests with argo spec: "TestListRuns_Multiuser"

func TestValidateCreateRunRequest(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &api.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	err := server.validateCreateRunRequest(&api.CreateRunRequest{Run: run})
	assert.Nil(t, err)
}

func TestValidateCreateRunRequest_WithPipelineVersionReference(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &api.Run{
		Name:               "run1",
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
	}
	err := server.validateCreateRunRequest(&api.CreateRunRequest{Run: run})
	assert.Nil(t, err)
}

func TestValidateCreateRunRequest_EmptyName(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &api.Run{
		ResourceReferences: validReference,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	err := server.validateCreateRunRequest(&api.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "The run name is empty")
}

func TestValidateCreateRunRequest_InvalidPipelineVersionReference(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &api.Run{
		Name:               "run1",
		ResourceReferences: referencesOfExperimentAndInvalidPipelineVersion,
	}
	err := server.validateCreateRunRequest(&api.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Get pipelineVersionId failed.")
}

func TestValidateCreateRunRequest_NoExperiment(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &api.Run{
		Name:               "run1",
		ResourceReferences: nil,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	err := server.validateCreateRunRequest(&api.CreateRunRequest{Run: run})
	assert.Nil(t, err)
}

func TestValidateCreateRunRequest_NilPipelineSpecAndEmptyPipelineVersion(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &api.Run{
		Name:               "run1",
		ResourceReferences: validReference,
	}
	err := server.validateCreateRunRequest(&api.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please specify a pipeline by providing a (workflow manifest or pipeline manifest) or (pipeline id or/and pipeline version).")
}

func TestValidateCreateRunRequest_WorkflowManifestAndPipelineVersion(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &api.Run{
		Name:               "run1",
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	err := server.validateCreateRunRequest(&api.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please don't specify a pipeline version or pipeline ID when you specify a workflow manifest or pipeline manifest.")
}

func TestValidateCreateRunRequest_InvalidPipelineSpec(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})
	run := &api.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &api.PipelineSpec{
			PipelineId:       resource.DefaultFakeUUID,
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	err := server.validateCreateRunRequest(&api.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please don't specify a pipeline version or pipeline ID when you specify a workflow manifest or pipeline manifest.")
}

func TestValidateCreateRunRequest_TooMuchParameters(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager, &RunServerOptions{CollectMetrics: false})

	var params []*api.Parameter
	// Create a long enough parameter string so it exceed the length limit of parameter.
	for i := 0; i < 10000; i++ {
		params = append(params, &api.Parameter{Name: "param2", Value: "world"})
	}
	run := &api.Run{
		Name:               "run1",
		ResourceReferences: validReference,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       params,
		},
	}

	err := server.validateCreateRunRequest(&api.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "The input parameter length exceed maximum size")
}

func TestReportRunMetrics_RunNotFound(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager, resourceManager, _ := initWithOneTimeRun(t)
	defer clientManager.Close()
	runServer := RunServer{resourceManager: resourceManager, options: &RunServerOptions{CollectMetrics: false}}

	_, err := runServer.ReportRunMetrics(context.Background(), &api.ReportRunMetricsRequest{
		RunId: "1",
	})
	AssertUserError(t, err, codes.NotFound)
}

func TestReportRunMetrics_Succeed(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager, resourceManager, runDetails := initWithOneTimeRun(t)
	defer clientManager.Close()
	runServer := RunServer{resourceManager: resourceManager, options: &RunServerOptions{CollectMetrics: false}}

	metric := &api.RunMetric{
		Name:   "metric-1",
		NodeId: "node-1",
		Value: &api.RunMetric_NumberValue{
			NumberValue: 0.88,
		},
		Format: api.RunMetric_RAW,
	}
	response, err := runServer.ReportRunMetrics(context.Background(), &api.ReportRunMetricsRequest{
		RunId:   runDetails.UUID,
		Metrics: []*api.RunMetric{metric},
	})
	assert.Nil(t, err)
	expectedResponse := &api.ReportRunMetricsResponse{
		Results: []*api.ReportRunMetricsResponse_ReportRunMetricResult{
			{
				MetricName:   metric.Name,
				MetricNodeId: metric.NodeId,
				Status:       api.ReportRunMetricsResponse_ReportRunMetricResult_OK,
			},
		},
	}
	assert.Equal(t, expectedResponse, response)

	run, err := runServer.GetRun(context.Background(), &api.GetRunRequest{
		RunId: runDetails.UUID,
	})
	assert.Nil(t, err)
	assert.Equal(t, []*api.RunMetric{metric}, run.GetRun().GetMetrics())
}

func TestReportRunMetrics_PartialFailures(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager, resourceManager, runDetail := initWithOneTimeRun(t)
	defer clientManager.Close()
	runServer := RunServer{resourceManager: resourceManager, options: &RunServerOptions{CollectMetrics: false}}

	validMetric := &api.RunMetric{
		Name:   "metric-1",
		NodeId: "node-1",
		Value: &api.RunMetric_NumberValue{
			NumberValue: 0.88,
		},
		Format: api.RunMetric_RAW,
	}
	invalidNameMetric := &api.RunMetric{
		Name:   "$metric-1",
		NodeId: "node-1",
	}
	invalidNodeIDMetric := &api.RunMetric{
		Name: "metric-1",
	}
	response, err := runServer.ReportRunMetrics(context.Background(), &api.ReportRunMetricsRequest{
		RunId:   runDetail.UUID,
		Metrics: []*api.RunMetric{validMetric, invalidNameMetric, invalidNodeIDMetric},
	})
	assert.Nil(t, err)
	expectedResponse := &api.ReportRunMetricsResponse{
		Results: []*api.ReportRunMetricsResponse_ReportRunMetricResult{
			{
				MetricName:   validMetric.Name,
				MetricNodeId: validMetric.NodeId,
				Status:       api.ReportRunMetricsResponse_ReportRunMetricResult_OK,
			},
			{
				MetricName:   invalidNameMetric.Name,
				MetricNodeId: invalidNameMetric.NodeId,
				Status:       api.ReportRunMetricsResponse_ReportRunMetricResult_INVALID_ARGUMENT,
			},
			{
				MetricName:   invalidNodeIDMetric.Name,
				MetricNodeId: invalidNodeIDMetric.NodeId,
				Status:       api.ReportRunMetricsResponse_ReportRunMetricResult_INVALID_ARGUMENT,
			},
		},
	}
	// Message fields, which are not reliable, are ignored from the test.
	for _, result := range response.Results {
		result.Message = ""
	}
	assert.Equal(t, expectedResponse, response)
}

// Removed tests with old auth spec: "TestCanAccessRun_Unauthorized", "TestCanAccessRun_Authorized", "TestCanAccessRun_Unauthenticated"
