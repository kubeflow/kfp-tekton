// Copyright 2018 The Kubeflow Authors
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

package main

import (
	"flag"
	"fmt"
	"strings"
	"time"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/crd/controller/scheduledworkflow/util"
	swfclientset "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned"
	swfinformers "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/informers/externalversions"
	"github.com/kubeflow/pipelines/backend/src/crd/pkg/signals"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	masterURL                   string
	kubeconfig                  string
	namespace                   string
	location                    *time.Location
	clientQPS                   float64
	clientBurst                 int
	executionType               string
	initializeTimeout           time.Duration
	timeout                     time.Duration
	mlPipelineAPIServerName     string
	mlPipelineAPIServerPort     string
	mlPipelineAPIServerBasePath string
	mlPipelineServiceHttpPort   string
	mlPipelineServiceGRPCPort   string
)

const (
	initializationTimeoutFlagName       = "initializeTimeout"
	timeoutFlagName                     = "timeout"
	mlPipelineAPIServerBasePathFlagName = "mlPipelineAPIServerBasePath"
	mlPipelineAPIServerNameFlagName     = "mlPipelineAPIServerName"
	mlPipelineAPIServerHttpPortFlagName = "mlPipelineServiceHttpPort"
	mlPipelineAPIServerGRPCPortFlagName = "mlPipelineServiceGRPCPort"
	addressTemp                         = "%s:%s"
)

func main() {
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	// Use the commonutil to store the ExecutionType
	commonutil.SetExecutionType(commonutil.ExecutionType(executionType))

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %s", err.Error())
	}
	cfg.QPS = float32(clientQPS)
	cfg.Burst = clientBurst

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	scheduleClient, err := swfclientset.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error building schedule clientset: %s", err.Error())
	}

	runServiceClient, err := NewRunServiceclient()
	if err != nil {
		log.Fatalf("Error connecting to apiserver: %s", err.Error())
	}

	clientParam := commonutil.ClientParameters{QPS: float64(cfg.QPS), Burst: cfg.Burst}
	execClient := commonutil.NewExecutionClientOrFatal(commonutil.CurrentExecutionType(), time.Second*30, clientParam)

	var scheduleInformerFactory swfinformers.SharedInformerFactory
	execInformer := commonutil.NewExecutionInformerOrFatal(commonutil.CurrentExecutionType(), namespace, time.Second*30, clientParam)
	if namespace == "" {
		scheduleInformerFactory = swfinformers.NewSharedInformerFactory(scheduleClient, time.Second*30)
	} else {
		scheduleInformerFactory = swfinformers.NewFilteredSharedInformerFactory(scheduleClient, time.Second*30, namespace, nil)
	}

	controller := NewController(
		kubeClient,
		scheduleClient,
		execClient,
		scheduleInformerFactory,
		execInformer,
		runServiceClient,
		commonutil.NewRealTime(),
		location)

	go scheduleInformerFactory.Start(stopCh)
	go execInformer.InformerFactoryStart(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		log.Fatalf("Error running controller: %s", err.Error())
	}
}

func NewRunServiceclient() (apiv2.RunServiceClient, error) {

	httpAddress := fmt.Sprintf(addressTemp, mlPipelineAPIServerName, mlPipelineServiceHttpPort)
	grpcAddress := fmt.Sprintf(addressTemp, mlPipelineAPIServerName, mlPipelineServiceGRPCPort)
	err := commonutil.WaitForAPIAvailable(initializeTimeout, mlPipelineAPIServerBasePath, httpAddress)
	if err != nil {
		return nil, errors.Wrapf(err,
			"Failed to initialize pipeline client. Error: %s", err.Error())
	}
	connection, err := commonutil.GetRpcConnection(grpcAddress)
	if err != nil {
		return nil, errors.Wrapf(err,
			"Failed to get RPC connection. Error: %s", err.Error())
	}
	return apiv2.NewRunServiceClient(connection), nil
}

func initEnv() {
	// Import environment variable, support nested vars e.g. OBJECTSTORECONFIG_ACCESSKEY
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()
	viper.AllowEmptyEnv(true)
}

func init() {
	initEnv()

	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&namespace, "namespace", "", "The namespace name used for Kubernetes informers to obtain the listers.")
	// Use default value of client QPS (5) & burst (10) defined in
	// k8s.io/client-go/rest/config.go#RESTClientFor
	flag.Float64Var(&clientQPS, "clientQPS", 5, "The maximum QPS to the master from this client.")
	flag.IntVar(&clientBurst, "clientBurst", 10, "Maximum burst for throttle from this client.")
	flag.StringVar(&executionType, "executionType", "Workflow", "Custom Resource's name of the backend Orchestration Engine")
	flag.DurationVar(&initializeTimeout, initializationTimeoutFlagName, 2*time.Minute, "Duration to wait for initialization of the ML pipeline API server.")
	flag.DurationVar(&timeout, timeoutFlagName, 1*time.Minute, "Duration to wait for calls to complete.")
	flag.StringVar(&mlPipelineAPIServerName, mlPipelineAPIServerNameFlagName, "ml-pipeline", "Name of the ML pipeline API server.")
	flag.StringVar(&mlPipelineServiceHttpPort, mlPipelineAPIServerHttpPortFlagName, "8888", "Http Port of the ML pipeline API server.")
	flag.StringVar(&mlPipelineServiceGRPCPort, mlPipelineAPIServerGRPCPortFlagName, "8887", "GRPC Port of the ML pipeline API server.")
	flag.StringVar(&mlPipelineAPIServerBasePath, mlPipelineAPIServerBasePathFlagName,
		"/apis/v2beta1", "The base path for the ML pipeline API server.")

	var err error
	location, err = util.GetLocation()
	if err != nil {
		log.Fatalf("Error running controller: %s", err.Error())
	}
	log.Infof("Location: %s", location.String())
}
