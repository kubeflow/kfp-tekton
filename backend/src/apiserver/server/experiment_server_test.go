package server

import (
	"context"
	"strings"
	"testing"

	"github.com/golang/protobuf/ptypes/timestamp"
	api "github.com/kubeflow/pipelines/backend/api/v1/go_client"
	kfpauth "github.com/kubeflow/pipelines/backend/src/apiserver/auth"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	authorizationv1 "k8s.io/api/authorization/v1"
)

func TestListExperiment_Unauthenticated(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{"no-identity-header": "user"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()

	server := ExperimentServer{manager, &ExperimentServerOptions{CollectMetrics: false}}
	_, err := server.ListExperiment(ctx, &api.ListExperimentsRequest{
		ResourceReferenceKey: &api.ResourceKey{
			Type: api.ResourceType_NAMESPACE,
			Id:   "ns1",
		},
	})
	assert.NotNil(t, err)
	assert.EqualError(
		t,
		err,
		wrapFailedAuthzApiResourcesError(wrapFailedAuthzApiResourcesError(kfpauth.IdentityHeaderMissingError)).Error(),
	)
}

func TestCreateExperiment(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &api.Experiment{Name: "ex1", Description: "first experiment"}

	result, err := server.CreateExperiment(nil, &api.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	expectedExperiment := &api.Experiment{
		Id:           resource.DefaultFakeUUID,
		Name:         "ex1",
		Description:  "first experiment",
		CreatedAt:    &timestamp.Timestamp{Seconds: 1},
		StorageState: api.Experiment_STORAGESTATE_AVAILABLE,
	}
	assert.Equal(t, expectedExperiment, result)
}

func TestCreateExperiment_Failed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &api.Experiment{Name: "ex1", Description: "first experiment"}
	clientManager.DB().Close()
	_, err := server.CreateExperiment(nil, &api.CreateExperimentRequest{Experiment: experiment})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Create experiment failed.")
}

func TestCreateExperiment_SingleUser_NamespaceNotAllowed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	resourceReferences := []*api.ResourceReference{
		{
			Key:          &api.ResourceKey{Type: api.ResourceType_NAMESPACE, Id: "ns1"},
			Relationship: api.Relationship_OWNER,
		},
	}
	experiment := &api.Experiment{
		Name:               "exp1",
		Description:        "first experiment",
		ResourceReferences: resourceReferences,
	}

	_, err := server.CreateExperiment(nil, &api.CreateExperimentRequest{Experiment: experiment})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "In single-user mode, CreateExperimentRequest shouldn't contain resource references.")
}

func TestCreateExperiment_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, resourceManager, _ := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &api.Experiment{
		Name:        "exp1",
		Description: "first experiment",
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_NAMESPACE, Id: "ns1"},
				Relationship: api.Relationship_OWNER,
			},
		}}

	_, err := server.CreateExperiment(ctx, &api.CreateExperimentRequest{Experiment: experiment})
	assert.NotNil(t, err)
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: "ns1",
		Verb:      common.RbacResourceVerbCreate,
		Group:     common.RbacPipelinesGroup,
		Version:   common.RbacPipelinesVersion,
		Resource:  common.RbacResourceTypeExperiments,
		Name:      experiment.Name,
	}
	assert.EqualError(
		t,
		err,
		wrapFailedAuthzRequestError(wrapFailedAuthzApiResourcesError(getPermissionDeniedError(userIdentity, resourceAttributes))).Error(),
	)
}

func TestCreateExperiment_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	resourceReferences := []*api.ResourceReference{
		{
			Key:          &api.ResourceKey{Type: api.ResourceType_NAMESPACE, Id: "ns1"},
			Relationship: api.Relationship_OWNER,
		},
	}
	experiment := &api.Experiment{
		Name:               "exp1",
		Description:        "first experiment",
		ResourceReferences: resourceReferences,
	}

	result, err := server.CreateExperiment(ctx, &api.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	expectedExperiment := &api.Experiment{
		Id:                 resource.DefaultFakeUUID,
		Name:               "exp1",
		Description:        "first experiment",
		CreatedAt:          &timestamp.Timestamp{Seconds: 1},
		ResourceReferences: resourceReferences,
		StorageState:       api.Experiment_STORAGESTATE_AVAILABLE,
	}
	assert.Equal(t, expectedExperiment, result)
}

func TestGetExperiment(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &api.Experiment{Name: "ex1", Description: "first experiment"}

	createResult, err := server.CreateExperiment(nil, &api.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	result, err := server.GetExperiment(nil, &api.GetExperimentRequest{Id: createResult.Id})
	assert.Nil(t, err)
	expectedExperiment := &api.Experiment{
		Id:           createResult.Id,
		Name:         "ex1",
		Description:  "first experiment",
		CreatedAt:    &timestamp.Timestamp{Seconds: 1},
		StorageState: api.Experiment_STORAGESTATE_AVAILABLE,
	}
	assert.Equal(t, expectedExperiment, result)
}

func TestGetExperiment_Failed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &api.Experiment{Name: "ex1", Description: "first experiment"}

	createResult, err := server.CreateExperiment(nil, &api.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	clientManager.DB().Close()
	_, err = server.GetExperiment(nil, &api.GetExperimentRequest{Id: createResult.Id})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Get experiment failed.")
}

func TestGetExperiment_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, experiment := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := ExperimentServer{manager, &ExperimentServerOptions{CollectMetrics: false}}

	_, err := server.GetExperiment(ctx, &api.GetExperimentRequest{Id: experiment.UUID})
	assert.NotNil(t, err)
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: "ns1",
		Verb:      common.RbacResourceVerbGet,
		Group:     common.RbacPipelinesGroup,
		Version:   common.RbacPipelinesVersion,
		Resource:  common.RbacResourceTypeExperiments,
		Name:      "exp1",
	}
	assert.EqualError(
		t,
		err,
		wrapFailedAuthzRequestError(wrapFailedAuthzApiResourcesError(getPermissionDeniedError(userIdentity, resourceAttributes))).Error(),
	)
}

func TestGetExperiment_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	resourceReferences := []*api.ResourceReference{
		{
			Key:          &api.ResourceKey{Type: api.ResourceType_NAMESPACE, Id: "ns1"},
			Relationship: api.Relationship_OWNER,
		},
	}
	experiment := &api.Experiment{
		Name:               "exp1",
		Description:        "first experiment",
		ResourceReferences: resourceReferences,
	}

	createResult, err := server.CreateExperiment(ctx, &api.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	result, err := server.GetExperiment(ctx, &api.GetExperimentRequest{Id: createResult.Id})
	assert.Nil(t, err)
	expectedExperiment := &api.Experiment{
		Id:                 createResult.Id,
		Name:               "exp1",
		Description:        "first experiment",
		CreatedAt:          &timestamp.Timestamp{Seconds: 1},
		ResourceReferences: resourceReferences,
		StorageState:       api.Experiment_STORAGESTATE_AVAILABLE,
	}
	assert.Equal(t, expectedExperiment, result)
}

func TestListExperiment(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &api.Experiment{Name: "ex1", Description: "first experiment"}

	createResult, err := server.CreateExperiment(nil, &api.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	result, err := server.ListExperiment(nil, &api.ListExperimentsRequest{})
	expectedExperiment := []*api.Experiment{{
		Id:           createResult.Id,
		Name:         "ex1",
		Description:  "first experiment",
		CreatedAt:    &timestamp.Timestamp{Seconds: 1},
		StorageState: api.Experiment_STORAGESTATE_AVAILABLE,
	}}
	assert.Nil(t, err)
	assert.Equal(t, expectedExperiment, result.Experiments)
}

func TestListExperiment_Failed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &api.Experiment{Name: "ex1", Description: "first experiment"}

	_, err := server.CreateExperiment(nil, &api.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	clientManager.DB().Close()
	_, err = server.ListExperiment(nil, &api.ListExperimentsRequest{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "List experiments failed.")
}

func TestListExperiment_SingleUser_NamespaceNotAllowed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager, options: &ExperimentServerOptions{CollectMetrics: false}}
	experiment := &api.Experiment{Name: "ex1", Description: "first experiment"}

	_, err := server.CreateExperiment(nil, &api.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	_, err = server.ListExperiment(nil, &api.ListExperimentsRequest{
		ResourceReferenceKey: &api.ResourceKey{
			Type: api.ResourceType_NAMESPACE,
			Id:   "ns1",
		},
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "In single-user mode, ListExperiment cannot filter by namespace.")
}

func TestListExperiment_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := ExperimentServer{manager, &ExperimentServerOptions{CollectMetrics: false}}

	_, err := server.ListExperiment(ctx, &api.ListExperimentsRequest{
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
		Resource:  common.RbacResourceTypeExperiments,
	}
	assert.EqualError(
		t,
		err,
		wrapFailedAuthzApiResourcesError(wrapFailedAuthzApiResourcesError(getPermissionDeniedError(userIdentity, resourceAttributes))).Error(),
	)
}

// Removed "TestListExperiment_Multiuser" test since it was using old k8s auth (0.11)

func TestValidateCreateExperimentRequest(t *testing.T) {
	tests := []struct {
		name         string
		experiment   *api.Experiment
		wantError    bool
		errorMessage string
	}{
		{
			"Valid",
			&api.Experiment{Name: "exp1", Description: "first experiment"},
			false,
			"",
		},
		{
			"Empty name",
			&api.Experiment{Description: "first experiment"},
			true,
			"name is empty",
		},
	}

	for _, tc := range tests {
		err := ValidateCreateExperimentRequest(&api.CreateExperimentRequest{Experiment: tc.experiment})
		if !tc.wantError && err != nil {
			t.Errorf("TestValidateCreateExperimentRequest(%v) expect no error but got %v", tc.name, err)
		}
		if tc.wantError {
			if err == nil {
				t.Errorf("TestValidateCreateExperimentRequest(%v) expect error but got nil", tc.name)
			} else if !strings.Contains(err.Error(), tc.errorMessage) {
				t.Errorf("TestValidateCreateExperimentRequest(%v) expect error containing: %v, but got: %v", tc.name, tc.errorMessage, err)
			}
		}
	}
}

func TestValidateCreateExperimentRequest_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	tests := []struct {
		name         string
		experiment   *api.Experiment
		wantError    bool
		errorMessage string
	}{
		{
			"Valid",
			&api.Experiment{
				Name:        "exp1",
				Description: "first experiment",
				ResourceReferences: []*api.ResourceReference{
					{
						Key:          &api.ResourceKey{Type: api.ResourceType_NAMESPACE, Id: "ns1"},
						Relationship: api.Relationship_OWNER,
					},
				},
			},
			false,
			"",
		},
		{
			"Missing namespace",
			&api.Experiment{
				Name:        "exp1",
				Description: "first experiment",
			},
			true,
			"Invalid resource references for experiment.",
		},
		{
			"Empty namespace",
			&api.Experiment{
				Name:        "exp1",
				Description: "first experiment",
				ResourceReferences: []*api.ResourceReference{
					{
						Key:          &api.ResourceKey{Type: api.ResourceType_NAMESPACE, Id: ""},
						Relationship: api.Relationship_OWNER,
					},
				},
			},
			true,
			"Invalid resource references for experiment. Namespace is empty.",
		},
		{
			"Multiple namespace",
			&api.Experiment{
				Name:        "exp1",
				Description: "first experiment",
				ResourceReferences: []*api.ResourceReference{
					{
						Key:          &api.ResourceKey{Type: api.ResourceType_NAMESPACE, Id: "ns1"},
						Relationship: api.Relationship_OWNER,
					},
					{
						Key:          &api.ResourceKey{Type: api.ResourceType_NAMESPACE, Id: "ns2"},
						Relationship: api.Relationship_OWNER,
					},
				},
			},
			true,
			"Invalid resource references for experiment.",
		},
		{
			"Invalid resource type",
			&api.Experiment{
				Name:        "exp1",
				Description: "first experiment",
				ResourceReferences: []*api.ResourceReference{
					{
						Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: "exp2"},
						Relationship: api.Relationship_OWNER,
					},
				},
			},
			true,
			"Invalid resource references for experiment.",
		},
	}

	for _, tc := range tests {
		err := ValidateCreateExperimentRequest(&api.CreateExperimentRequest{Experiment: tc.experiment})
		if !tc.wantError && err != nil {
			t.Errorf("TestValidateCreateExperimentRequest(%v) expect no error but got %v", tc.name, err)
		}
		if tc.wantError {
			if err == nil {
				t.Errorf("TestValidateCreateExperimentRequest(%v) expect error but got nil", tc.name)
			} else if !strings.Contains(err.Error(), tc.errorMessage) {
				t.Errorf("TestValidateCreateExperimentRequest(%v) expect error containing: %v, but got: %v", tc.name, tc.errorMessage, err)
			}
		}
	}
}

// remove "TestArchiveAndUnarchiveExperiment" test
