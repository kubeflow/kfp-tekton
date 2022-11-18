package server

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	api "github.com/kubeflow/pipelines/backend/api/v1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
)

// Removed tarball tests that check for argo yaml and old auth dep: "TestDecompressPipelineTarball", "TestDecompressPipelineTarball_NonYamlTarball",
// "TestDecompressPipelineZip", "TestDecompressPipelineZip_NonYamlZip", "TestDecompressPipelineZip_EmptyZip", "TestReadPipelineFile_YAML",
// "TestReadPipelineFile_Zip", "TestReadPipelineFile_Zip_AnyExtension", "TestReadPipelineFile_MultifileZip", "TestReadPipelineFile_Tarball",
// "TestReadPipelineFile_Tarball_AnyExtension", "TestReadPipelineFile_MultifileTarball", "TestReadPipelineFile_UnknownFileFormat",
// "TestValidatePipelineSpecAndResourceReferences_PipelineIdNotParentOfPipelineVersionId"

func TestReadPipelineFile_JSON(t *testing.T) {
	file, _ := os.Open("test/v2-hello-world.json")
	fileBytes, err := ReadPipelineFile("v2-hello-world.json", file, MaxFileLength)
	assert.Nil(t, err)

	expectedFileBytes, _ := ioutil.ReadFile("test/v2-hello-world.json")
	assert.Equal(t, expectedFileBytes, fileBytes)
}

func TestGetPipelineName_QueryStringNotEmpty(t *testing.T) {
	pipelineName, err := GetPipelineName("pipeline%20one", "file one")
	assert.Nil(t, err)
	assert.Equal(t, "pipeline one", pipelineName)
}

func TestGetPipelineName(t *testing.T) {
	pipelineName, err := GetPipelineName("", "file one")
	assert.Nil(t, err)
	assert.Equal(t, "file one", pipelineName)
}

func TestGetPipelineName_InvalidQueryString(t *testing.T) {
	_, err := GetPipelineName("pipeline!$%one", "file one")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "invalid format")
}

func TestGetPipelineName_NameTooLong(t *testing.T) {
	_, err := GetPipelineName("",
		"this is a loooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooog name")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "name too long")
}

func TestLoadFile(t *testing.T) {
	file := "12345"
	bytes, err := loadFile(strings.NewReader(file), 5)
	assert.Nil(t, err)
	assert.Equal(t, []byte(file), bytes)
}

func TestLoadFile_ExceedSizeLimit(t *testing.T) {
	file := "12345"
	_, err := loadFile(strings.NewReader(file), 4)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "File size too large")
}

// removed tests (check top page comment)

func TestDecompressPipelineTarball_MalformattedTarball(t *testing.T) {
	tarballByte, _ := ioutil.ReadFile("test/malformatted_tarball.tar.gz")
	_, err := DecompressPipelineTarball(tarballByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Not a valid tarball file")
}

// removed tests (check top page comment)

func TestDecompressPipelineTarball_EmptyTarball(t *testing.T) {
	tarballByte, _ := ioutil.ReadFile("test/empty_tarball/empty.tar.gz")
	_, err := DecompressPipelineTarball(tarballByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Not a valid tarball file")
}

// removed tests (check top page comment)

func TestDecompressPipelineZip_MalformattedZip(t *testing.T) {
	zipByte, _ := ioutil.ReadFile("test/malformatted_zip.zip")
	_, err := DecompressPipelineZip(zipByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Not a valid zip file")
}

func TestDecompressPipelineZip_MalformedZip2(t *testing.T) {
	zipByte, _ := ioutil.ReadFile("test/malformed_zip2.zip")
	_, err := DecompressPipelineZip(zipByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Not a valid zip file")
}

// removed tests (check top page comment)

func TestValidateExperimentResourceReference(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	assert.Nil(t, ValidateExperimentResourceReference(manager, validReference))
}

func TestValidateExperimentResourceReference_MoreThanOneRef(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	references := []*api.ResourceReference{
		{
			Key: &api.ResourceKey{
				Type: api.ResourceType_EXPERIMENT, Id: "123"},
			Relationship: api.Relationship_OWNER,
		},
		{
			Key: &api.ResourceKey{
				Type: api.ResourceType_EXPERIMENT, Id: "456"},
			Relationship: api.Relationship_OWNER,
		},
	}
	err := ValidateExperimentResourceReference(manager, references)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "more resource references than expected")
}

func TestValidateExperimentResourceReference_UnexpectedType(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	references := []*api.ResourceReference{
		{
			Key: &api.ResourceKey{
				Type: api.ResourceType_UNKNOWN_RESOURCE_TYPE, Id: "123"},
			Relationship: api.Relationship_OWNER,
		},
	}
	err := ValidateExperimentResourceReference(manager, references)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unexpected resource type")
}

func TestValidateExperimentResourceReference_EmptyID(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	references := []*api.ResourceReference{
		{
			Key: &api.ResourceKey{
				Type: api.ResourceType_EXPERIMENT},
			Relationship: api.Relationship_OWNER,
		},
	}
	err := ValidateExperimentResourceReference(manager, references)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Resource ID is empty")
}

func TestValidateExperimentResourceReference_UnexpectedRelationship(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	references := []*api.ResourceReference{
		{
			Key: &api.ResourceKey{
				Type: api.ResourceType_EXPERIMENT, Id: "123"},
			Relationship: api.Relationship_CREATOR,
		},
	}
	err := ValidateExperimentResourceReference(manager, references)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unexpected relationship for the experiment")
}

func TestValidateExperimentResourceReference_ExperimentNotExist(t *testing.T) {
	clients := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := resource.NewResourceManager(clients)
	defer clients.Close()
	err := ValidateExperimentResourceReference(manager, validReference)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to get experiment")
}

func TestValidatePipelineSpecAndResourceReferences_WorkflowManifestAndPipelineVersion(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	spec := &api.PipelineSpec{
		WorkflowManifest: testWorkflow.ToStringForStore()}
	err := ValidatePipelineSpecAndResourceReferences(manager, spec, validReferencesOfExperimentAndPipelineVersion)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please don't specify a pipeline version or pipeline ID when you specify a workflow manifest or pipeline manifest.")
}

func TestValidatePipelineSpecAndResourceReferences_WorkflowManifestAndPipelineID(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	spec := &api.PipelineSpec{
		PipelineId:       resource.DefaultFakeUUID,
		WorkflowManifest: testWorkflow.ToStringForStore()}
	err := ValidatePipelineSpecAndResourceReferences(manager, spec, validReference)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please don't specify a pipeline version or pipeline ID when you specify a workflow manifest or pipeline manifest.")
}

func TestValidatePipelineSpecAndResourceReferences_InvalidWorkflowManifest(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	spec := &api.PipelineSpec{WorkflowManifest: "I am an invalid manifest"}
	err := ValidatePipelineSpecAndResourceReferences(manager, spec, validReference)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Invalid argo workflow format.")
}

func TestValidatePipelineSpecAndResourceReferences_NilPipelineSpecAndEmptyPipelineVersion(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	err := ValidatePipelineSpecAndResourceReferences(manager, nil, validReference)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please specify a pipeline by providing a (workflow manifest or pipeline manifest) or (pipeline id or/and pipeline version).")
}

func TestValidatePipelineSpecAndResourceReferences_EmptyPipelineSpecAndEmptyPipelineVersion(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	spec := &api.PipelineSpec{}
	err := ValidatePipelineSpecAndResourceReferences(manager, spec, validReference)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please specify a pipeline by providing a (workflow manifest or pipeline manifest) or (pipeline id or/and pipeline version).")
}

func TestValidatePipelineSpecAndResourceReferences_InvalidPipelineId(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	spec := &api.PipelineSpec{PipelineId: "not-found"}
	err := ValidatePipelineSpecAndResourceReferences(manager, spec, validReference)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Get pipelineId failed.")
}

func TestValidatePipelineSpecAndResourceReferences_InvalidPipelineVersionId(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	err := ValidatePipelineSpecAndResourceReferences(manager, nil, referencesOfInvalidPipelineVersion)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Get pipelineVersionId failed.")
}

func TestValidatePipelineSpecAndResourceReferences_ParameterTooLongWithPipelineId(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	var params []*api.Parameter
	// Create a long enough parameter string so it exceed the length limit of parameter.
	for i := 0; i < 10000; i++ {
		params = append(params, &api.Parameter{Name: "param2", Value: "world"})
	}
	spec := &api.PipelineSpec{PipelineId: resource.DefaultFakeUUID, Parameters: params}
	err := ValidatePipelineSpecAndResourceReferences(manager, spec, validReference)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "The input parameter length exceed maximum size")
}

func TestValidatePipelineSpecAndResourceReferences_ParameterTooLongWithWorkflowManifest(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	var params []*api.Parameter
	// Create a long enough parameter string so it exceed the length limit of parameter.
	for i := 0; i < 10000; i++ {
		params = append(params, &api.Parameter{Name: "param2", Value: "world"})
	}
	spec := &api.PipelineSpec{WorkflowManifest: testWorkflow.ToStringForStore(), Parameters: params}
	err := ValidatePipelineSpecAndResourceReferences(manager, spec, validReference)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "The input parameter length exceed maximum size")
}

func TestValidatePipelineSpecAndResourceReferences_ValidPipelineIdAndPipelineVersionId(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	spec := &api.PipelineSpec{
		PipelineId: resource.DefaultFakeUUID}
	err := ValidatePipelineSpecAndResourceReferences(manager, spec, validReferencesOfExperimentAndPipelineVersion)
	assert.Nil(t, err)
}

func TestValidatePipelineSpecAndResourceReferences_ValidWorkflowManifest(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	spec := &api.PipelineSpec{WorkflowManifest: testWorkflow.ToStringForStore()}
	err := ValidatePipelineSpecAndResourceReferences(manager, spec, validReference)
	assert.Nil(t, err)
}
