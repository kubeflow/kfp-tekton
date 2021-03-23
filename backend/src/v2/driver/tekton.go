package main

import (
	"context"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"github.com/pkg/errors"

	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	tektoncdclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/typed/pipeline/v1beta1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

const (
	DefaultConnectionTimeout = "6m"
)

// TektonInterface provides functions to get Tekton Task and update Tekton Task
type TektonInterface interface {
	GetTask(ctx context.Context, name string) (*v1beta1.Task, error)
	UpdateTask(ctx context.Context, task *v1beta1.Task) (*v1beta1.Task, error)
	PatchTaskContainer(ctx context.Context, name string, stepIdx int, container k8sv1.Container) (*v1beta1.Task, error)
}

// TektonClientTask is the client
type TektonClientTask struct {
	tektonV1beta1Task tektonv1beta1.TaskInterface
}

// GetTask get the specified tekton task
func (c TektonClientTask) GetTask(ctx context.Context, name string) (*v1beta1.Task, error) {
	return c.tektonV1beta1Task.Get(ctx, name, metav1.GetOptions{})
}

// UpdateTask update the task
func (c TektonClientTask) UpdateTask(ctx context.Context, task *v1beta1.Task) (*v1beta1.Task, error) {
	return c.tektonV1beta1Task.Update(ctx, task, metav1.UpdateOptions{})
}

// PatchTaskContainer patch the task named 'name' with the specified Container to a specific step
func (c TektonClientTask) PatchTaskContainer(ctx context.Context, name string, stepIdx int,
	container k8sv1.Container) (*v1beta1.Task, error) {

	task, err := c.GetTask(ctx, taskTemplateName)
	if err != nil {
		return nil, err
	}

	if stepIdx > len(task.Spec.Steps) {
		return nil, errors.Errorf("incorrect step index: %v", stepIdx)
	}
	task.Spec.Steps[stepIdx].Container = container
	return c.UpdateTask(ctx, task)
}

func createTektonClient(namespace string) (TektonInterface, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize kubernetes client.")
	}

	clientSet, err := tektoncdclientset.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize kubernetes client set.")
	}
	return TektonClientTask{clientSet.TektonV1beta1().Tasks(namespace)}, nil
}

// CreateTektonClientOrFatal creates a new client for the Kubernetes pod.
func CreateTektonClientOrFatal(namespace string) TektonInterface {
	var client TektonInterface
	var err error
	var operation = func() error {
		client, err = createTektonClient(namespace)
		if err != nil {
			return err
		}
		return nil
	}
	b := backoff.NewExponentialBackOff()
	timeoutDuration, _ := time.ParseDuration(DefaultConnectionTimeout)
	b.MaxElapsedTime = timeoutDuration
	err = backoff.Retry(operation, b)

	if err != nil {
		glog.Fatalf("Failed to create tekton client. Error: %v", err)
	}
	return client
}
