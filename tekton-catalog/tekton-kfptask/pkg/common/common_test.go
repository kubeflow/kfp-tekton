package common

import (
	"testing"

	"github.com/kubeflow/pipelines/backend/src/v2/driver"

	"github.com/stretchr/testify/assert"
)

func TestUpdateOptionsDAGExecutionID(t *testing.T) {
	options := &driverOptions{
		options: driver.Options{},
	}

	UpdateOptionsDAGExecutionID(options, "456")

	assert.Equal(t, int64(456), options.options.DAGExecutionID)
}

func TestUpdateOptionsIterationIndex(t *testing.T) {
	options := &driverOptions{
		options: driver.Options{},
	}

	UpdateOptionsIterationIndex(options, 42)

	assert.Equal(t, 42, options.options.IterationIndex)
}

func TestPrettyPrint(t *testing.T) {
	jsonStr := `{"key": "value"}`

	result := prettyPrint(jsonStr)

	assert.NotEmpty(t, result)
}
