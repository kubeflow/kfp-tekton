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

	"github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/apis/pipelineloop"
	pipelineloopv1alpha1 "github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/apis/pipelineloop/v1alpha1"
	pipelineloopclient "github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/client/injection/client"
	pipelineloopinformer "github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/client/injection/informers/pipelineloop/v1alpha1/pipelineloop"
	cl "github.com/scrapcodes/cos-logger"
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
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/system"
)

func load(ctx context.Context, kubeClientSet kubernetes.Interface, o *cl.ObjectStoreLogConfig) error {
	configMap, err := kubeClientSet.CoreV1().ConfigMaps(system.Namespace()).
		Get(ctx, "object-store-config", metaV1.GetOptions{})
	if err != nil {
		return err
	}
	if o.Enable, err = strconv.ParseBool(configMap.Data["enable"]); err != nil || !o.Enable {
		return err
	}

	o.AccessKey = configMap.Data["accessKey"]
	o.SecretKey = configMap.Data["secretKey"]
	o.Region = configMap.Data["region"]
	o.ServiceEndpoint = configMap.Data["serviceEndpoint"]
	o.DefaultBucketName = configMap.Data["defaultBucketName"]
	o.CreateBucket = false
	o.Token = configMap.Data["token"]
	return nil
}

// NewController instantiates a new controller.Impl from knative.dev/pkg/controller
func NewController(namespace string) func(context.Context, configmap.Watcher) *controller.Impl {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		kubeclientset := kubeclient.Get(ctx)
		logger := logging.FromContext(ctx)
		pipelineclientset := pipelineclient.Get(ctx)
		pipelineloopclientset := pipelineloopclient.Get(ctx)
		runInformer := runinformer.Get(ctx)
		pipelineLoopInformer := pipelineloopinformer.Get(ctx)
		pipelineRunInformer := pipelineruninformer.Get(ctx)

		c := &Reconciler{
			KubeClientSet:         kubeclientset,
			pipelineClientSet:     pipelineclientset,
			pipelineloopClientSet: pipelineloopclientset,
			runLister:             runInformer.Lister(),
			pipelineLoopLister:    pipelineLoopInformer.Lister(),
			pipelineRunLister:     pipelineRunInformer.Lister(),
		}
		loggerConfig := cl.ObjectStoreLogConfig{}
		objectStoreLogger := cl.Logger{
			MaxSize: 1024 * 100, // TODO make it configurable via a configmap.
		}
		err := load(ctx, kubeclientset, &loggerConfig)
		if err == nil {
			err = objectStoreLogger.LoadDefaults(loggerConfig)
			if err == nil {
				_ = objectStoreLogger.LogConfig.CreateNewBucket(loggerConfig.DefaultBucketName)
			}
		}
		if err == nil && objectStoreLogger.LogConfig.Enable {
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
			logger := zap.New(core)
			logger.Info("First log msg with object store logger.")
			ctx = logging.WithLogger(ctx, logger.Sugar())

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

		impl := runreconciler.NewImpl(ctx, c, func(impl *controller.Impl) controller.Options {
			return controller.Options{
				AgentName: "run-pipelineloop",
			}
		})

		logger.Info("Setting up event handlers")

		// Add event handler for Runs
		runInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: pipelinecontroller.FilterRunRef(pipelineloopv1alpha1.SchemeGroupVersion.String(), pipelineloop.PipelineLoopControllerName),
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
