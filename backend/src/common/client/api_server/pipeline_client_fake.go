package api_server

import (
	"fmt"

	"path"

	"github.com/go-openapi/strfmt"
	params "github.com/kubeflow/pipelines/backend/api/v1/go_http_client/pipeline_client/pipeline_service"
	pipelineparams "github.com/kubeflow/pipelines/backend/api/v1/go_http_client/pipeline_client/pipeline_service"
	pipelinemodel "github.com/kubeflow/pipelines/backend/api/v1/go_http_client/pipeline_model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	workflowapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Replaced Argo v1alpha1.Workflow to Tekton v1beta1.PipelineRun

const (
	PipelineForDefaultTest     = "PIPELINE_ID_10"
	PipelineForClientErrorTest = "PIPELINE_ID_11"
	PipelineValidURL           = "http://www.mydomain.com/foo.yaml"
	PipelineInvalidURL         = "foobar.something"
)

func getDefaultPipeline(id string) *pipelinemodel.V1Pipeline {
	return &pipelinemodel.V1Pipeline{
		CreatedAt:   strfmt.NewDateTime(),
		Description: "PIPELINE_DESCRIPTION",
		ID:          id,
		Name:        "PIPELINE_NAME",
		Parameters: []*pipelinemodel.V1Parameter{&pipelinemodel.V1Parameter{
			Name:  "PARAM_NAME",
			Value: "PARAM_VALUE",
		}},
	}
}

func getDefaultWorkflow() *workflowapi.PipelineRun {
	return &workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
		}}
}

func getDefaultTemplate() template.Template {
	tmpl, _ := template.NewTektonTemplateFromWorkflow(&workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
		}})
	return tmpl
}

func getDefaultWorkflowAsString() string {
	tmpl := getDefaultTemplate()
	return string(tmpl.Bytes())
}

type PipelineClientFake struct{}

func NewPipelineClientFake() *PipelineClientFake {
	return &PipelineClientFake{}
}

func (c *PipelineClientFake) Create(params *pipelineparams.PipelineServiceCreatePipelineParams) (
	*pipelinemodel.V1Pipeline, error) {
	switch params.Pipeline.URL.PipelineURL {
	case PipelineInvalidURL:
		return nil, fmt.Errorf(ClientErrorString)
	default:
		return getDefaultPipeline(path.Base(params.Pipeline.URL.PipelineURL)), nil
	}
}

func (c *PipelineClientFake) Get(params *pipelineparams.PipelineServiceGetPipelineParams) (
	*pipelinemodel.V1Pipeline, error) {
	switch params.ID {
	case PipelineForClientErrorTest:
		return nil, fmt.Errorf(ClientErrorString)
	default:
		return getDefaultPipeline(params.ID), nil
	}
}

func (c *PipelineClientFake) Delete(params *pipelineparams.PipelineServiceDeletePipelineParams) error {
	switch params.ID {
	case PipelineForClientErrorTest:
		return fmt.Errorf(ClientErrorString)
	default:
		return nil
	}
}

func (c *PipelineClientFake) GetTemplate(params *pipelineparams.PipelineServiceGetTemplateParams) (
	template.Template, error) {
	switch params.ID {
	case PipelineForClientErrorTest:
		return nil, fmt.Errorf(ClientErrorString)
	default:
		return getDefaultTemplate(), nil
	}
}

func (c *PipelineClientFake) List(params *pipelineparams.PipelineServiceListPipelinesParams) (
	[]*pipelinemodel.V1Pipeline, int, string, error) {

	const (
		FirstToken  = ""
		SecondToken = "SECOND_TOKEN"
		FinalToken  = ""
	)

	token := ""
	if params.PageToken != nil {
		token = *params.PageToken
	}

	switch token {
	case FirstToken:
		return []*pipelinemodel.V1Pipeline{
			getDefaultPipeline("PIPELINE_ID_100"),
			getDefaultPipeline("PIPELINE_ID_101"),
		}, 2, SecondToken, nil
	case SecondToken:
		return []*pipelinemodel.V1Pipeline{
			getDefaultPipeline("PIPELINE_ID_102"),
		}, 1, FinalToken, nil
	default:
		return nil, 0, "", fmt.Errorf(InvalidFakeRequest, token)
	}
}

func (c *PipelineClientFake) ListAll(params *pipelineparams.PipelineServiceListPipelinesParams,
	maxResultSize int) ([]*pipelinemodel.V1Pipeline, error) {
	return listAllForPipeline(c, params, maxResultSize)
}

func (c *PipelineClientFake) UpdateDefaultVersion(params *params.PipelineServiceUpdatePipelineDefaultVersionParams) error {
	switch params.PipelineID {
	case PipelineForClientErrorTest:
		return fmt.Errorf(ClientErrorString)
	default:
		return nil
	}
}
