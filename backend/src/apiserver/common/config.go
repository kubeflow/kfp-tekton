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
	"strconv"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
)

const (
	MultiUserMode                       string = "MULTIUSER"
	PodNamespace                        string = "POD_NAMESPACE"
	CacheEnabled                        string = "CacheEnabled"
	DefaultPipelineRunnerServiceAccount string = "DefaultPipelineRunnerServiceAccount"
	KubeflowUserIDHeader                string = "KUBEFLOW_USERID_HEADER"
	KubeflowUserIDPrefix                string = "KUBEFLOW_USERID_PREFIX"
	ArchiveLogs                         string = "ARCHIVE_LOGS"
	TrackArtifacts                      string = "TRACK_ARTIFACTS"
	StripEOF                            string = "STRIP_EOF"
	ArtifactBucket                      string = "ARTIFACT_BUCKET"
	ArtifactEndpoint                    string = "ARTIFACT_ENDPOINT"
	ArtifactEndpointScheme              string = "ARTIFACT_ENDPOINT_SCHEME"
	ArtifactScript                      string = "ARTIFACT_SCRIPT"
	ArtifactImage                       string = "ARTIFACT_IMAGE"
	ArtifactCopyStepTemplate            string = "ARTIFACT_COPY_STEP_TEMPLATE"
	InjectDefaultScript                 string = "INJECT_DEFAULT_SCRIPT"
)

func GetStringConfig(configName string) string {
	if !viper.IsSet(configName) {
		glog.Fatalf("Please specify flag %s", configName)
	}
	return viper.GetString(configName)
}

func GetStringConfigWithDefault(configName, value string) string {
	if !viper.IsSet(configName) {
		return value
	}
	return viper.GetString(configName)
}

func GetMapConfig(configName string) map[string]string {
	if !viper.IsSet(configName) {
		glog.Infof("Config %s not specified, skipping", configName)
		return nil
	}
	return viper.GetStringMapString(configName)
}

func GetBoolConfigWithDefault(configName string, value bool) bool {
	if !viper.IsSet(configName) {
		return value
	}
	value, err := strconv.ParseBool(viper.GetString(configName))
	if err != nil {
		glog.Fatalf("Failed converting string to bool %s", viper.GetString(configName))
	}
	return value
}

func GetDurationConfig(configName string) time.Duration {
	if !viper.IsSet(configName) {
		glog.Fatalf("Please specify flag %s", configName)
	}
	return viper.GetDuration(configName)
}

func IsMultiUserMode() bool {
	return GetBoolConfigWithDefault(MultiUserMode, false)
}

func IsArchiveLogs() bool {
	return GetBoolConfigWithDefault(ArchiveLogs, false)
}

func IsStripEOF() bool {
	return GetBoolConfigWithDefault(StripEOF, true)
}

func IsTrackArtifacts() bool {
	return GetBoolConfigWithDefault(TrackArtifacts, true)
}

func IsInjectDefaultScript() bool {
	return GetBoolConfigWithDefault(InjectDefaultScript, true)
}

func GetPodNamespace() string {
	return GetStringConfig(PodNamespace)
}

func GetArtifactImage() string {
	return GetStringConfigWithDefault(ArtifactImage, DefaultArtifactImage)
}

func GetCopyStepTemplate() *corev1.Container {
	var tpl corev1.Container
	if err := viper.UnmarshalKey(ArtifactCopyStepTemplate, &tpl); err != nil {
		glog.Fatalf("Invalid '%s', %v", ArtifactCopyStepTemplate, err)
	}
	return &tpl
}

func GetBoolFromStringWithDefault(value string, defaultValue bool) bool {
	boolVal, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return boolVal
}

func IsCacheEnabled() string {
	return GetStringConfigWithDefault(CacheEnabled, "true")
}

func GetKubeflowUserIDHeader() string {
	return GetStringConfigWithDefault(KubeflowUserIDHeader, GoogleIAPUserIdentityHeader)
}

func GetKubeflowUserIDPrefix() string {
	return GetStringConfigWithDefault(KubeflowUserIDPrefix, GoogleIAPUserIdentityPrefix)
}

func GetArtifactBucket() string {
	return GetStringConfigWithDefault(ArtifactBucket, DefaultArtifactBucket)
}

func GetArtifactEndpoint() string {
	return GetStringConfigWithDefault(ArtifactEndpoint, DefaultArtifactEndpoint)
}

func GetArtifactEndpointScheme() string {
	return GetStringConfigWithDefault(ArtifactEndpointScheme, DefaultArtifactEndpointScheme)
}

func GetArtifactScript() string {
	return GetStringConfigWithDefault(ArtifactScript, DefaultArtifactScript)
}
