package template

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/ghodss/yaml"
	workflowapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	scheduledworkflow "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"k8s.io/apimachinery/pkg/types"
)

func (t *Tekton) RunWorkflow(apiRun *api.Run, options RunWorkflowOptions, namespace string) (*util.Workflow, error) {
	workflow := util.NewWorkflow(t.wf.PipelineRun.DeepCopy())

	// Add a KFP specific label for cache service filtering. The cache_enabled flag here is a global control for whether cache server will
	// receive targeting pods. Since cache server only receives pods in step level, the resource manager here will set this global label flag
	// on every single step/pod so the cache server can understand.
	if strings.ToLower(common.IsCacheEnabled()) != "true" {
		workflow.SetLabels(util.LabelKeyCacheEnabled, common.IsCacheEnabled())
	}

	// Add a KFP specific label for cache service filtering. The cache_enabled flag here is a global control for whether cache server will
	// receive targeting pods. Since cache server only receives pods in step level, the resource manager here will set this global label flag
	// on every single step/pod so the cache server can understand.
	// TODO: Add run_level flag with similar logic by reading flag value from create_run api.
	workflow.SetLabelsToAllTemplates(util.LabelKeyCacheEnabled, common.IsCacheEnabled())
	parameters := toParametersMap(apiRun.GetPipelineSpec().GetParameters())
	// Verify no additional parameter provided
	if err := workflow.VerifyParameters(parameters); err != nil {
		return nil, util.Wrap(err, "Failed to verify parameters.")
	}
	// Append provided parameter
	workflow.OverrideParameters(parameters)

	// Replace macros
	formatter := util.NewRunParameterFormatter(options.RunId, options.RunAt)
	formattedParams := formatter.FormatWorkflowParameters(workflow.GetWorkflowParametersAsMap())
	workflow.OverrideParameters(formattedParams)

	setDefaultServiceAccount(workflow, apiRun.GetServiceAccount())

	// Disable istio sidecar injection if not specified
	workflow.SetAnnotationsToAllTemplatesIfKeyNotExist(util.AnnotationKeyIstioSidecarInject, util.AnnotationValueIstioSidecarInjectDisabled)

	err := OverrideParameterWithSystemDefault(workflow)
	if err != nil {
		return nil, err
	}

	// Add label to the workflow so it can be persisted by persistent agent later.
	workflow.SetLabels(util.LabelKeyWorkflowRunId, options.RunId)
	// Add run name annotation to the workflow so that it can be logged by the Metadata Writer.
	workflow.SetAnnotations(util.AnnotationKeyRunName, apiRun.Name)
	// Replace {{workflow.uid}} and $(context.pipelineRun.uid) with runId
	err = workflow.ReplaceUID(options.RunId)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to replace workflow ID")
	}
	workflow.Name = workflow.Name + "-" + options.RunId[0:5]

	// Add original label to the workflow so it can be persisted by persistent agent later.
	workflow.SetLabels(util.LabelOriginalPipelineRunName, workflow.Name)
	err = workflow.ReplaceOrignalPipelineRunName(workflow.Name)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to replace workflow original pipelineRun name")
	}

	// Predefine custom resource if resource_templates are provided and feature flag
	// is enabled.
	if strings.ToLower(common.IsApplyTektonCustomResource()) == "true" {
		if tektonTemplates, ok := workflow.Annotations["tekton.dev/resource_templates"]; ok {
			err = t.applyCustomResources(*workflow, tektonTemplates, namespace)
			if err != nil {
				return nil, util.NewInternalServerError(err, "Apply Tekton Custom resource Failed")
			}
		}
	}

	err = t.tektonPreprocessing(*workflow)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Tekton Preprocessing Failed")
	}

	return workflow, nil

}

type Tekton struct {
	wf *util.Workflow
}

func (t *Tekton) ScheduledWorkflow(apiJob *api.Job, namespace string) (*scheduledworkflow.ScheduledWorkflow, error) {
	workflow := util.NewWorkflow(t.wf.PipelineRun.DeepCopy())

	parameters := toParametersMap(apiJob.GetPipelineSpec().GetParameters())
	// Verify no additional parameter provided
	if err := workflow.VerifyParameters(parameters); err != nil {
		return nil, util.Wrap(err, "Failed to verify parameters.")
	}
	// Append provided parameter
	workflow.OverrideParameters(parameters)
	setDefaultServiceAccount(workflow, apiJob.GetServiceAccount())
	// Disable istio sidecar injection if not specified
	workflow.SetAnnotations(util.AnnotationKeyIstioSidecarInject, util.AnnotationValueIstioSidecarInjectDisabled)
	swfGeneratedName, err := toSWFCRDResourceGeneratedName(apiJob.Name)
	if err != nil {
		return nil, util.Wrap(err, "Create job failed")
	}
	// Predefine custom resource if resource_templates are provided and feature flag
	// is enabled.
	if strings.ToLower(common.IsApplyTektonCustomResource()) == "true" {
		if tektonTemplates, ok := workflow.Annotations["tekton.dev/resource_templates"]; ok {
			err = t.applyCustomResources(*workflow, tektonTemplates, namespace)
			if err != nil {
				return nil, util.NewInternalServerError(err, "Apply Tekton Custom resource Failed")
			}
		}
	}

	err = t.tektonPreprocessing(*workflow)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Tekton Preprocessing Failed")
	}
	scheduledWorkflow := &scheduledworkflow.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{GenerateName: swfGeneratedName},
		Spec: scheduledworkflow.ScheduledWorkflowSpec{
			Enabled:        apiJob.Enabled,
			MaxConcurrency: &apiJob.MaxConcurrency,
			Trigger:        *toCRDTrigger(apiJob.Trigger),
			Workflow: &scheduledworkflow.WorkflowResource{
				Parameters: toCRDParameter(apiJob.GetPipelineSpec().GetParameters()),
				Spec:       workflow.Spec,
			},
			NoCatchup: util.BoolPointer(apiJob.NoCatchup),
		},
	}
	return scheduledWorkflow, nil
}

func (t *Tekton) GetTemplateType() TemplateType {
	return V1
}

func NewTektonTemplate(bytes []byte) (*Tekton, error) {
	wf, err := ValidatePipelineRun(bytes)
	if err != nil {
		return nil, err
	}
	return &Tekton{wf}, nil
}

func (t *Tekton) Bytes() []byte {
	if t == nil {
		return nil
	}
	return []byte(t.wf.ToStringForStore())
}

func (t *Tekton) IsV2() bool {
	if t == nil {
		return false
	}
	return t.wf.IsV2Compatible()
}

const (
	paramV2compatPipelineName = "pipeline-name"
)

func (t *Tekton) V2PipelineName() string {
	if t == nil {
		return ""
	}
	return t.wf.GetWorkflowParametersAsMap()[paramV2compatPipelineName]
}

func (t *Tekton) OverrideV2PipelineName(name, namespace string) {
	if t == nil || !t.wf.IsV2Compatible() {
		return
	}
	var pipelineRef string
	if namespace != "" {
		pipelineRef = fmt.Sprintf("namespace/%s/pipeline/%s", namespace, name)
	} else {
		pipelineRef = fmt.Sprintf("pipeline/%s", name)
	}
	overrides := make(map[string]string)
	overrides[paramV2compatPipelineName] = pipelineRef
	t.wf.OverrideParameters(overrides)
}

func (t *Tekton) ParametersJSON() (string, error) {
	if t == nil {
		return "", nil
	}
	return MarshalParameters(t.wf.Spec.Params)
}

func NewTektonTemplateFromWorkflow(wf *workflowapi.PipelineRun) (*Tekton, error) {
	return &Tekton{wf: &util.Workflow{PipelineRun: wf}}, nil
}

func ValidatePipelineRun(template []byte) (*util.Workflow, error) {
	var pr workflowapi.PipelineRun
	err := yaml.Unmarshal(template, &pr)
	if err != nil {
		return nil, util.NewInvalidInputErrorWithDetails(err, "Failed to parse the PipelineRun template.")
	}
	if pr.APIVersion != TektonVersion {
		return nil, util.NewInvalidInputError("Unsupported argo version. Expected: %v. Received: %v", TektonVersion, pr.APIVersion)
	}
	if pr.Kind != TektonK8sResource {
		return nil, util.NewInvalidInputError("Unexpected resource type. Expected: %v. Received: %v", TektonK8sResource, pr.Kind)
	}
	// TODO: Add Tekton validate
	return util.NewWorkflow(&pr), nil
}

// tektonPreprocessing injects artifacts and logging steps if it's enabled
func (t *Tekton) tektonPreprocessing(workflow util.Workflow) error {
	// Tekton: Update artifact cred using the KFP Tekton configmap
	workflow.SetAnnotations(common.ArtifactBucketAnnotation, common.GetArtifactBucket())
	workflow.SetAnnotations(common.ArtifactEndpointAnnotation, common.GetArtifactEndpoint())
	workflow.SetAnnotations(common.ArtifactEndpointSchemeAnnotation, common.GetArtifactEndpointScheme())

	// Process artifacts
	artifactItems, exists := workflow.ObjectMeta.Annotations[common.ArtifactItemsAnnotation]

	// Only inject artifacts if the necessary annotations are provided.
	if exists {
		var artifactItemsJSON map[string][][]interface{}
		if err := json.Unmarshal([]byte(artifactItems), &artifactItemsJSON); err != nil {
			return err
		}
		t.injectArchivalStep(workflow, artifactItemsJSON)
	}
	return nil
}

func (t *Tekton) injectArchivalStep(workflow util.Workflow, artifactItemsJSON map[string][][]interface{}) {
	for _, task := range workflow.Spec.PipelineSpec.Tasks {
		artifacts, hasArtifacts := artifactItemsJSON[task.Name]
		archiveLogs := common.IsArchiveLogs()
		trackArtifacts := common.IsTrackArtifacts()
		stripEOF := common.IsStripEOF()
		injectDefaultScript := common.IsInjectDefaultScript()
		copyStepTemplate := common.GetCopyStepTemplate()

		artifactAnnotation, exists := workflow.ObjectMeta.Annotations[common.TrackArtifactAnnotation]
		if exists && strings.ToLower(artifactAnnotation) == "true" {
			trackArtifacts = true
		}
		if task.TaskSpec != nil && task.TaskSpec.Steps != nil {
			injectStepArtifacts := false
			stepArtifactAnnotation, exists := task.TaskSpec.Metadata.Annotations[common.TrackStepArtifactAnnotation]
			if trackArtifacts || (exists && strings.ToLower(stepArtifactAnnotation) == "true") {
				injectStepArtifacts = true
			}
			if (hasArtifacts && len(artifacts) > 0 && injectStepArtifacts) || archiveLogs || (hasArtifacts && len(artifacts) > 0 && stripEOF) {
				artifactScript := common.GetArtifactScript()
				if archiveLogs {
					// Logging volumes
					if task.TaskSpec.Volumes == nil {
						task.TaskSpec.Volumes = []corev1.Volume{}
					}
					loggingVolumes := []corev1.Volume{
						t.getHostPathVolumeSource("varlog", "/var/log"),
						t.getHostPathVolumeSource("varlibdockercontainers", "/var/lib/docker/containers"),
						t.getHostPathVolumeSource("varlibkubeletpods", "/var/lib/kubelet/pods"),
						t.getHostPathVolumeSource("varlogpods", "/var/log/pods"),
					}
					task.TaskSpec.Volumes = append(task.TaskSpec.Volumes, loggingVolumes...)

					// Logging volumeMounts
					if task.TaskSpec.StepTemplate == nil {
						task.TaskSpec.StepTemplate = &workflowapi.StepTemplate{}
					}
					if task.TaskSpec.StepTemplate.VolumeMounts == nil {
						task.TaskSpec.StepTemplate.VolumeMounts = []corev1.VolumeMount{}
					}
					loggingVolumeMounts := []corev1.VolumeMount{
						{Name: "varlog", MountPath: "/var/log"},
						{Name: "varlibdockercontainers", MountPath: "/var/lib/docker/containers", ReadOnly: true},
						{Name: "varlibkubeletpods", MountPath: "/var/lib/kubelet/pods", ReadOnly: true},
						{Name: "varlogpods", MountPath: "/var/log/pods", ReadOnly: true},
					}
					task.TaskSpec.StepTemplate.VolumeMounts = append(task.TaskSpec.StepTemplate.VolumeMounts, loggingVolumeMounts...)
				}

				moveStep := false
				if task.TaskSpec.Results != nil && len(task.TaskSpec.Results) > 0 {
					// move all results to /tekton/home/tep-results to avoid result duplication in copy-artifacts step
					// TODO: disable eof strip, since no results under /tekton/results after this step
					moveResults := workflowapi.Step{
						Image:   common.GetMoveResultsImage(),
						Name:    "move-all-results-to-tekton-home",
						Command: []string{"sh", "-c"},
						Args: []string{fmt.Sprintf("if [ -d /tekton/results ]; then mkdir -p %s; mv /tekton/results/* %s/ || true; fi\n",
							common.GetPath4InternalResults(), common.GetPath4InternalResults())},
					}
					moveStep = true
					task.TaskSpec.Steps = append(task.TaskSpec.Steps, moveResults)
				}

				// Process the artifacts into minimum sh commands if running with minimum linux kernel
				if injectDefaultScript {
					artifactScript = t.injectDefaultScript(workflow, artifactScript, artifacts, hasArtifacts, archiveLogs, injectStepArtifacts, stripEOF, moveStep)
				}

				// Define post-processing step
				step := *copyStepTemplate
				if step.Name == "" {
					step.Name = "copy-artifacts"
				}
				if step.Image == "" {
					step.Image = common.GetArtifactImage()
				}
				step.Env = append(step.Env,
					t.getObjectFieldSelector("ARTIFACT_BUCKET", "metadata.annotations['tekton.dev/artifact_bucket']"),
					t.getObjectFieldSelector("ARTIFACT_ENDPOINT", "metadata.annotations['tekton.dev/artifact_endpoint']"),
					t.getObjectFieldSelector("ARTIFACT_ENDPOINT_SCHEME", "metadata.annotations['tekton.dev/artifact_endpoint_scheme']"),
					t.getObjectFieldSelector("ARTIFACT_ITEMS", "metadata.annotations['tekton.dev/artifact_items']"),
					t.getObjectFieldSelector("PIPELINETASK", "metadata.labels['tekton.dev/pipelineTask']"),
					t.getObjectFieldSelector("PIPELINERUN", "metadata.labels['tekton.dev/pipelineRun']"),
					t.getObjectFieldSelector("PODNAME", "metadata.name"),
					t.getObjectFieldSelector("NAMESPACE", "metadata.namespace"),
					t.getSecretKeySelector("AWS_ACCESS_KEY_ID", "mlpipeline-minio-artifact", "accesskey"),
					t.getSecretKeySelector("AWS_SECRET_ACCESS_KEY", "mlpipeline-minio-artifact", "secretkey"),
					t.getEnvVar("ARCHIVE_LOGS", strconv.FormatBool(archiveLogs)),
					t.getEnvVar("TRACK_ARTIFACTS", strconv.FormatBool(trackArtifacts)),
					t.getEnvVar("STRIP_EOF", strconv.FormatBool(stripEOF)),
				)
				step.Command = []string{"sh", "-c"}
				step.Args = []string{artifactScript}

				task.TaskSpec.Steps = append(task.TaskSpec.Steps, step)
			}
		}
	}
}

func (t *Tekton) injectDefaultScript(workflow util.Workflow, artifactScript string,
	artifacts [][]interface{}, hasArtifacts, archiveLogs, trackArtifacts, stripEOF, hasMoveStep bool) string {
	// Need to represent as Raw String Literals
	artifactScript += "\n"
	if archiveLogs {
		artifactScript += "push_log\n"
	}

	// Upload Artifacts if the artifact is enabled and the annoations are present
	if hasArtifacts && len(artifacts) > 0 && trackArtifacts {
		for _, artifact := range artifacts {
			artifactBaseName := fmt.Sprintf("\"$(basename \"%s\")", artifact[1])
			artifactPathName := fmt.Sprintf("\"%s/%s", common.GetPath4InternalResults(), artifactBaseName)
			if artifact[0] == "mlpipeline-ui-metadata" || artifact[0] == "mlpipeline-metrics" {
				artifactPathName = fmt.Sprintf("\"%s\"", artifact[1])
			}
			if len(artifact) == 2 {
				artifactScript += fmt.Sprintf("push_artifact \"%s\" %s\n",
					artifact[0], artifactPathName)
			} else {
				glog.Warningf("Artifact annotations are missing for run %v.", workflow.Name)
			}
		}
	}

	// Strip EOF if enabled, do it after artifact upload since it only applies to parameter outputs
	if !hasMoveStep && hasArtifacts && len(artifacts) > 0 && stripEOF {
		for _, artifact := range artifacts {
			if len(artifact) == 2 {
				// The below solution is in experimental stage and didn't cover all edge cases.
				artifactScript += fmt.Sprintf("strip_eof %s %s\n", artifact[0], artifact[1])
			} else {
				glog.Warningf("Artifact annotations are missing for run %v.", workflow.Name)
			}
		}
	}
	return artifactScript
}

func (t *Tekton) getObjectFieldSelector(name string, fieldPath string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: name,
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: fieldPath,
			},
		},
	}
}

func (t *Tekton) getSecretKeySelector(name string, objectName string, objectKey string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: name,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: objectName,
				},
				Key: objectKey,
			},
		},
	}
}

func (t *Tekton) getEnvVar(name string, value string) corev1.EnvVar {
	return corev1.EnvVar{
		Name:  name,
		Value: value,
	}
}

func (t *Tekton) getHostPathVolumeSource(name string, path string) corev1.Volume {
	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: path,
			},
		},
	}
}

func (t *Tekton) applyCustomResources(workflow util.Workflow, tektonTemplates string, namespace string) error {
	// Create kubeClient to deploy Tekton custom task crd
	var config *rest.Config
	config, err := rest.InClusterConfig()
	if err != nil {
		glog.Errorf("error creating client configuration: %v", err)
		return err
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Errorf("Failed to create client: %v", err)
		return err
	}
	var templates []interface{}
	// Decode metadata into JSON payload.
	err = json.Unmarshal([]byte(tektonTemplates), &templates)
	if err != nil {
		glog.Errorf("Failed to Unmarshal custom task CRD: %v", err)
		return err
	}
	for i := range templates {
		template := templates[i]
		apiVersion, ok := template.(map[string]interface{})["apiVersion"].(string)
		if !ok {
			glog.Errorf("Failed to get Tekton custom task apiVersion")
			return errors.New("Failed to get Tekton custom task apiVersion")
		}
		singlarKind, ok := template.(map[string]interface{})["kind"].(string)
		if !ok {
			glog.Errorf("Failed to get Tekton custom task kind")
			return errors.New("Failed to get Tekton custom task kind")
		}
		api := strings.Split(apiVersion, "/")[0]
		version := strings.Split(apiVersion, "/")[1]
		resource := strings.ToLower(singlarKind) + "s"
		name, ok := template.(map[string]interface{})["metadata"].(map[string]interface{})["name"].(string)
		if !ok {
			glog.Errorf("Failed to get Tekton custom task name")
			return errors.New("Failed to get Tekton custom task name")
		}
		body, err := json.Marshal(template)
		if err != nil {
			glog.Errorf("Failed to convert to JSON: %v", err)
			return err
		}
		// Check whether the resource is exist, if yes do a patch
		// if not do a post(create)
		_, err = kubeClient.RESTClient().
			Get().
			AbsPath("/apis/" + api + "/" + version).
			Namespace(namespace).
			Resource(resource).
			Name(name).
			DoRaw(context.Background())
		if err != nil {
			_, err = kubeClient.RESTClient().Post().
				AbsPath(fmt.Sprintf("/apis/%s/%s", api, version)).
				Namespace(namespace).
				Resource(resource).
				Body(body).
				DoRaw(context.Background())
			if err != nil {
				glog.Errorf("Failed to create resource for pipeline: %s, %v", workflow.Name, err)
				return err
			}
		} else {
			_, err = kubeClient.RESTClient().Patch(types.MergePatchType).
				AbsPath(fmt.Sprintf("/apis/%s/%s", api, version)).
				Namespace(namespace).
				Resource(resource).
				Name(name).
				Body(body).
				DoRaw(context.Background())
			if err != nil {
				glog.Errorf("Failed to patch resource for pipeline: %s, %v", workflow.Name, err)
				return err
			}
		}
	}
	return nil
}
