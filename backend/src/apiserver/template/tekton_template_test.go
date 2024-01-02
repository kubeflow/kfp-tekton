package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestGetConfigMapVolumeSource(t *testing.T) {
	t.Run("Returns correct volume source", func(t *testing.T) {
		// Create a new instance of Tekton
		tekton := &Tekton{}

		// Define the expected volume source
		expectedVolumeSource := corev1.Volume{
			Name: "testVolume",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "testConfigMap",
					},
				},
			},
		}

		// Call the getConfigMapVolumeSource method
		volumeSource := tekton.getConfigMapVolumeSource("testVolume", "testConfigMap")

		// Assert that the returned volume source matches the expected volume source
		assert.Equal(t, expectedVolumeSource, volumeSource)
	})
}
func TestGetHostPathVolumeSource(t *testing.T) {
	t.Run("Returns correct volume source", func(t *testing.T) {
		// Create a new instance of Tekton
		tekton := &Tekton{}

		// Define the expected volume source
		expectedVolumeSource := corev1.Volume{
			Name: "testVolume",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "testPath",
				},
			},
		}

		// Call the getHostPathVolumeSource method
		volumeSource := tekton.getHostPathVolumeSource("testVolume", "testPath")

		// Assert that the returned volume source matches the expected volume source
		assert.Equal(t, expectedVolumeSource, volumeSource)
	})
}

func TestGetEnvVar(t *testing.T) {
	t.Run("Returns correct environment variable", func(t *testing.T) {
		// Create a new instance of Tekton
		tekton := &Tekton{}

		// Define the input parameters
		name := "testName"
		value := "testValue"

		// Call the getEnvVar method
		envVar := tekton.getEnvVar(name, value)

		// Assert that the returned environment variable has the correct name and value
		assert.Equal(t, name, envVar.Name)
		assert.Equal(t, value, envVar.Value)
	})
}

func TestGetSecretKeySelector(t *testing.T) {
	t.Run("Returns correct environment variable", func(t *testing.T) {
		// Create a new instance of Tekton
		tekton := &Tekton{}

		// Define the input parameters
		name := "testName"
		objectName := "testObjectName"
		objectKey := "testObjectKey"

		// Call the getSecretKeySelector method
		envVar := tekton.getSecretKeySelector(name, objectName, objectKey)

		// Assert that the returned environment variable has the correct name
		assert.Equal(t, name, envVar.Name)

		// Assert that the returned environment variable has the correct SecretKeyRef
		assert.NotNil(t, envVar.ValueFrom.SecretKeyRef)
		assert.Equal(t, objectName, envVar.ValueFrom.SecretKeyRef.LocalObjectReference.Name)
		assert.Equal(t, objectKey, envVar.ValueFrom.SecretKeyRef.Key)
	})
}

func TestGetObjectFieldSelector(t *testing.T) {
	t.Run("Returns correct environment variable", func(t *testing.T) {
		// Create a new instance of Tekton
		tekton := &Tekton{}

		// Define the input parameters
		name := "testName"
		fieldPath := "testFieldPath"

		// Call the getObjectFieldSelector method
		envVar := tekton.getObjectFieldSelector(name, fieldPath)

		// Assert that the returned environment variable has the correct name
		assert.Equal(t, name, envVar.Name)

		// Assert that the returned environment variable has the correct ObjectFieldSelector
		assert.NotNil(t, envVar.ValueFrom.FieldRef)
		assert.Equal(t, fieldPath, envVar.ValueFrom.FieldRef.FieldPath)
	})
}
