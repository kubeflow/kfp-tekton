/*
Copyright 2020 kubeflow.org.

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

package main

import (
	"context"
	"flag"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	tektoncdclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/apis"
)

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == strings.TrimSpace(str) {
			return true
		}
	}
	return false
}

func main() {
	var namespace string
	var prName string
	var taskList string
	var failedTasks []string

	flag.StringVar(&namespace, "namespace", "", "The namespace of the pipelinerun.")
	flag.StringVar(&prName, "prName", "", "The name of the pipelinerun.")
	flag.StringVar(&taskList, "taskList", "", "The comma separated list of the tasks.")
	flag.Parse()

	for _, arg := range []string{namespace, prName, taskList} {
		if arg == "" {
			log.Errorf("The arguments namespace, prName and taskList are required.")
			os.Exit(1)
		}
	}

	log.Printf("Starting to watch taskrun for '%s' in %s/%s.", taskList, namespace, prName)

	tasks := strings.Split(taskList, ",")

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Errorf("Get config of the cluster failed: %+v", err)
		os.Exit(1)
	}

	tektonClient, err := tektoncdclientset.NewForConfig(config)
	if err != nil {
		log.Errorf("Get client of tekton failed: %+v", err)
		os.Exit(1)
	}

	labelSelector := "tekton.dev/pipelineRun=" + strings.TrimSpace(prName)

	for {
		watcher, err := tektonClient.TektonV1beta1().TaskRuns(namespace).Watch(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector})

		if err != nil {
			log.Printf("TaskRun Watcher error:" + err.Error())
			log.Printf("Please ensure the service account has permission to get taskRun.")
			os.Exit(1)
		}

		for event := range watcher.ResultChan() {
			taskrun := event.Object.(*v1beta1.TaskRun)
			taskLabel := taskrun.Labels["tekton.dev/pipelineTask"]
			if contains(tasks, taskLabel) {
				if taskrun.Status.GetCondition(apis.ConditionSucceeded).IsTrue() {
					log.Printf("The TaskRun of %s succeeded.", taskLabel)
					watcher.Stop()
					os.Exit(0)
				} else if taskrun.Status.GetCondition(apis.ConditionSucceeded).IsFalse() {
					if !contains(failedTasks, taskLabel) {
						failedTasks = append(failedTasks, taskLabel)
					}
					if len(failedTasks) >= len(tasks) {
						log.Printf("All specified TaskRun(s) failed.")
						watcher.Stop()
						os.Exit(1)
					}
				}
			}
		}
	}
}
