package client

import (
	"context"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"github.com/pkg/errors"

	// "k8s.io/client-go/kubernetes"
	// v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	tektoncdclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

type TektonInterface interface {
	GetTaskRun(namespace, name string, queryOptions metav1.GetOptions) (*v1beta1.TaskRun, error)
}

type TektonClient struct {
	tektonV1beta1Client *tektoncdclientset.Clientset
}

func (c TektonClient) GetTaskRun(namespace, name string, queryOptions metav1.GetOptions) (*v1beta1.TaskRun, error) {
	return c.tektonV1beta1Client.TektonV1beta1().TaskRuns(namespace).Get(context.Background(), name, queryOptions)
}

func createTektonClient() (TektonInterface, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize kubernetes client.")
	}

	clientSet, err := tektoncdclientset.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize kubernetes client set.")
	}
	return TektonClient{clientSet}, nil
}

// CreateTektonClientOrFatal creates a new client for the Kubernetes pod.
func CreateTektonClientOrFatal(initConnectionTimeout time.Duration) TektonInterface {
	var client TektonInterface
	var err error
	var operation = func() error {
		client, err = createTektonClient()
		if err != nil {
			return err
		}
		return nil
	}
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initConnectionTimeout
	err = backoff.Retry(operation, b)

	if err != nil {
		glog.Fatalf("Failed to create tekton client. Error: %v", err)
	}
	return client
}
