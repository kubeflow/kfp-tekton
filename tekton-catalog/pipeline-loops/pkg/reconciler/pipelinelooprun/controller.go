/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pipelinelooprun

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	backoff "github.com/cenkalti/backoff/v4"
	taskCache "github.com/kubeflow/kfp-tekton/tekton-catalog/cache/pkg"
	"github.com/kubeflow/kfp-tekton/tekton-catalog/cache/pkg/db"
	cl "github.com/kubeflow/kfp-tekton/tekton-catalog/objectstore/pkg/writer"
	"github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/apis/pipelineloop"
	pipelineloopv1alpha1 "github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/apis/pipelineloop/v1alpha1"
	pipelineloopclient "github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/client/injection/client"
	pipelineloopinformer "github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/client/injection/informers/pipelineloop/v1alpha1/pipelineloop"
	pipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client"
	runinformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1alpha1/run"
	pipelineruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/pipelinerun"
	runreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1alpha1/run"
	pipelinecontroller "github.com/tektoncd/pipeline/pkg/controller"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/clock"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/system"
)

func loadObjectStoreConfig(ctx context.Context, kubeClientSet kubernetes.Interface, o *cl.ObjectStoreConfig) (bool, error) {
	configMap, err := kubeClientSet.CoreV1().ConfigMaps(system.Namespace()).
		Get(ctx, "object-store-config", metaV1.GetOptions{})
	if err != nil {
		return false, err
	}
	var enable bool
	if enable, err = strconv.ParseBool(configMap.Data["enable"]); err != nil || !enable {
		return false, err
	}

	o.AccessKey = configMap.Data["accessKey"]
	o.SecretKey = configMap.Data["secretKey"]
	o.Region = configMap.Data["region"]
	o.ServiceEndpoint = configMap.Data["serviceEndpoint"]
	o.DefaultBucketName = configMap.Data["defaultBucketName"]
	o.CreateBucket = false
	o.Token = configMap.Data["token"]
	return true, nil
}

func loadCacheConfig(ctx context.Context, kubeClientSet kubernetes.Interface, p *db.ConnectionParams) (bool, error) {
	configMap, err := kubeClientSet.CoreV1().ConfigMaps(system.Namespace()).Get(ctx,
		"cache-config", metaV1.GetOptions{})
	if err != nil {
		return true, err
	}
	if configMap.Data["disabled"] == "true" {
		return true, nil
	}
	p.DbName = configMap.Data["dbName"]
	p.DbDriver = configMap.Data["driver"]
	p.DbHost = configMap.Data["host"]
	p.DbPort = configMap.Data["port"]
	p.DbExtraParams = configMap.Data["extraParams"]
	p.DbUser = configMap.Data["user"]
	p.DbPwd = configMap.Data["password"]
	timeout, err := time.ParseDuration(configMap.Data["timeout"])
	if err != nil {
		return true, fmt.Errorf("invalid value passed for timeout: %v", err)
	}
	p.Timeout = timeout
	return false, nil
}

var params db.ConnectionParams

func initCache(ctx context.Context, kubeClientSet kubernetes.Interface, params db.ConnectionParams) *taskCache.TaskCacheStore {
	logger := logging.FromContext(ctx)
	disabled, err := loadCacheConfig(ctx, kubeClientSet, &params)
	if err != nil {
		logger.Errorf("ConfigMap cache-config could not be loaded. "+
			"Cache store disabled. Error : %v", err)
	}
	cacheStore := &taskCache.TaskCacheStore{Params: params}
	if disabled {
		cacheStore.Disabled = true
	}
	if !cacheStore.Disabled {
		logger.Infof("Cache store Params: %#v", params)
		b := backoff.NewExponentialBackOff()
		b.MaxElapsedTime = params.Timeout
		var operation = func() error {
			return cacheStore.Connect()
		}
		err := backoff.Retry(operation, b)
		if err != nil {
			cacheStore.Disabled = true
			logger.Errorf("Failed to connect to cache store backend, cache store disabled. err: %v", err)
		} else {
			logger.Infof("Cache store connected to db with Params: %#v", params)
		}
	}
	return cacheStore
}

func initLogger(ctx context.Context, kubeClientSet kubernetes.Interface) *zap.SugaredLogger {
	var logger = logging.FromContext(ctx)
	loggerConfig := cl.ObjectStoreConfig{}
	objectStoreLogger := cl.Logger{
		MaxSize: 1024 * 100, // TODO make it configurable via a configmap.
	}
	enabled, err := loadObjectStoreConfig(ctx, kubeClientSet, &loggerConfig)
	if err == nil && enabled {
		err = objectStoreLogger.LoadDefaults(loggerConfig)
		if err == nil {
			_ = objectStoreLogger.Writer.CreateNewBucket(loggerConfig.DefaultBucketName)
		} else {
			logger.Errorf("error connecting to the object store, %v", err)
		}
	}
	if err == nil && enabled {
		logger.Info("Loading object store logger...")
		w := zapcore.NewMultiWriteSyncer(
			zapcore.AddSync(os.Stdout),
			zapcore.AddSync(&objectStoreLogger),
		)
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			w,
			zap.InfoLevel,
		)
		logger = zap.New(core).Sugar()
		logger.Info("First log msg with object store logger.")

		// set up SIGHUP to send logs to object store before shutdown.
		signal.Ignore(syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
		c := make(chan os.Signal, 3)
		signal.Notify(c, syscall.SIGTERM)
		signal.Notify(c, syscall.SIGINT)
		signal.Notify(c, syscall.SIGHUP)

		go func() {
			for {
				<-c
				err = objectStoreLogger.Close()
				fmt.Printf("Synced with object store... %v", err)
				os.Exit(0)
			}
		}()
	} else {
		logger.Errorf("Object store logging unavailable, %v ", err)
	}
	return logger
}

// NewController instantiates a new controller.Impl from knative.dev/pkg/controller
func NewController(namespace string) func(context.Context, configmap.Watcher) *controller.Impl {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		kubeclientset := kubeclient.Get(ctx)
		pipelineclientset := pipelineclient.Get(ctx)
		pipelineloopclientset := pipelineloopclient.Get(ctx)
		runInformer := runinformer.Get(ctx)
		pipelineLoopInformer := pipelineloopinformer.Get(ctx)
		pipelineRunInformer := pipelineruninformer.Get(ctx)
		logger := initLogger(ctx, kubeclientset)
		ctx = logging.WithLogger(ctx, logger)
		cacheStore := initCache(ctx, kubeclientset, params)
		c := &Reconciler{
			KubeClientSet:         kubeclientset,
			pipelineClientSet:     pipelineclientset,
			pipelineloopClientSet: pipelineloopclientset,
			runLister:             runInformer.Lister(),
			pipelineLoopLister:    pipelineLoopInformer.Lister(),
			pipelineRunLister:     pipelineRunInformer.Lister(),
			cacheStore:            cacheStore,
			clock:                 clock.RealClock{},
		}

		impl := runreconciler.NewImpl(ctx, c, func(impl *controller.Impl) controller.Options {
			return controller.Options{
				AgentName: "run-pipelineloop",
			}
		})

		logger.Info("Setting up event handlers")

		// Add event handler for Runs of Pipelineloop
		runInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: pipelinecontroller.FilterRunRef(pipelineloopv1alpha1.SchemeGroupVersion.String(), pipelineloop.PipelineLoopControllerName),
			Handler:    controller.HandleAll(impl.Enqueue),
		})
		// Add event handler for Runs of BreakTask
		runInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: pipelinecontroller.FilterRunRef(pipelineloopv1alpha1.SchemeGroupVersion.String(), pipelineloop.BreakTaskName),
			Handler:    controller.HandleAll(impl.Enqueue),
		})
		// Add event handler for PipelineRuns controlled by Run
		pipelineRunInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: pipelinecontroller.FilterOwnerRunRef(runInformer.Lister(), pipelineloopv1alpha1.SchemeGroupVersion.String(), pipelineloop.PipelineLoopControllerName),
			Handler:    controller.HandleAll(impl.EnqueueControllerOf),
		})

		return impl
	}
}
