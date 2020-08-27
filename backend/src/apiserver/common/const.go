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

package common

import (
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type ResourceType string
type Relationship string

const (
	Experiment      ResourceType = "Experiment"
	Job             ResourceType = "Job"
	Run             ResourceType = "Run"
	Pipeline        ResourceType = "pipeline"
	PipelineVersion ResourceType = "PipelineVersion"
	Namespace       ResourceType = "Namespace"
)

const (
	Owner   Relationship = "Owner"
	Creator Relationship = "Creator"
)

const (
	GoogleIAPUserIdentityHeader string = "x-goog-authenticated-user-email"
	GoogleIAPUserIdentityPrefix string = "accounts.google.com:"
)

const (
	DefaultArtifactBucket         string = "mlpipeline"
	DefaultArtifactEndpoint       string = "minio-service.kubeflow:9000"
	DefaultArtifactEndpointScheme string = "http://"
)

const (
	ArtifactItemsAnnotation          string = "tekton.dev/artifact_items"
	ArtifactBucketAnnotation         string = "tekton.dev/artifact_bucket"
	ArtifactEndpointAnnotation       string = "tekton.dev/artifact_endpoint"
	ArtifactEndpointSchemeAnnotation string = "tekton.dev/artifact_endpoint_scheme"
)

const DefaultArtifactScript string = "#!/usr/bin/env sh\n" +
	"push_artifact() {\n" +
	"    tar -cvzf $1.tgz $2\n" +
	"    mc cp $1.tgz storage/$ARTIFACT_BUCKET/artifacts/$PIPELINERUN/$PIPELINETASK/$1.tgz\n" +
	"}\n" +
	"strip_eof() {\n" +
	"    awk 'NF' $2 | head -c -1 > $1_temp_save && cp $1_temp_save $2\n" +
	"}\n" +
	"mc config host add storage ${ARTIFACT_ENDPOINT_SCHEME}${ARTIFACT_ENDPOINT} $AWS_ACCESS_KEY_ID $AWS_SECRET_ACCESS_KEY\n"

func ToModelResourceType(apiType api.ResourceType) (ResourceType, error) {
	switch apiType {
	case api.ResourceType_EXPERIMENT:
		return Experiment, nil
	case api.ResourceType_JOB:
		return Job, nil
	case api.ResourceType_PIPELINE_VERSION:
		return PipelineVersion, nil
	case api.ResourceType_NAMESPACE:
		return Namespace, nil
	default:
		return "", util.NewInvalidInputError("Unsupported resource type: %s", api.ResourceType_name[int32(apiType)])
	}
}

func ToModelRelationship(r api.Relationship) (Relationship, error) {
	switch r {
	case api.Relationship_CREATOR:
		return Creator, nil
	case api.Relationship_OWNER:
		return Owner, nil
	default:
		return "", util.NewInvalidInputError("Unsupported resource relationship: %s", api.Relationship_name[int32(r)])
	}
}
