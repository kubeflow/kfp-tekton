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

package cmd

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/Knetic/govaluate"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	tektoncdclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/apis"
)

var (
	namespace    string
	prName       string
	taskList     string
	conditions   []string
	conditionMap map[string][]conditionResult
)

type conditionResult struct {
	condition string
	results   []string
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == strings.TrimSpace(str) {
			return true
		}
	}
	return false
}

func parse_conditions(condtions []string, tasks *[]string) {
	conditionMap = make(map[string][]conditionResult)
	if len(condtions) == 0 {
		return
	}
	// Expect result to have the format of results_taskName_resultName
	for _, condition := range condtions {
		operands := regexp.MustCompile(" +").Split(condition, -1)
		if len(operands) != 3 {
			log.Printf("The conditon must be as format 'operand1 operator operand2'.")
			os.Exit(1)
		}
		var taskName string
		var resultName []string
		resultMatcher := regexp.MustCompile(`results_([^_]*)_([^_]+)`)
		operand1Results := resultMatcher.FindAllStringSubmatch(operands[0], -1)
		operand2Results := resultMatcher.FindAllStringSubmatch(operands[2], -1)
		if len(operand1Results) == 0 && len(operand2Results) == 0 {
			log.Printf("Must at least contain one result and at most two in one condition for a task.")
			os.Exit(1)
		}
		if len(operand1Results) > 0 && len(operand2Results) > 0 {
			if operand1Results[0][1] != operand2Results[0][1] {
				log.Printf("The conditon can only contain results in one task, here's two.")
				os.Exit(1)
			}
			taskName = operand1Results[0][1]
			resultName = []string{
				operand1Results[0][2],
				operand2Results[0][2],
			}
		}
		if len(operand1Results) > 0 {
			taskName = operand1Results[0][1]
			resultName = []string{
				operand1Results[0][2],
			}
		}
		if len(operand2Results) > 0 {
			taskName = operand2Results[0][1]
			resultName = []string{
				operand2Results[0][2],
			}
		}
		if !contains(*tasks, taskName) {
			*tasks = append(*tasks, taskName)
		}
		cr := conditionResult{
			condition: condition,
			results:   resultName,
		}
		if len(conditionMap[taskName]) == 0 {
			conditionMap[taskName] = []conditionResult{}
		}
		conditionMap[taskName] = append(conditionMap[taskName], cr)
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// rootCmd represents the base command when called without any subcommands
	var rootCmd = &cobra.Command{
		Use:   "any-taskrun",
		Short: "Watch taskrun and exit when any taskrun complete",
		Long: `Watch taskrun and exit when any of below is true:
			1: taskrun complete
			2: condition met`,
		Run: watch,
	}

	rootCmd.Flags().StringVar(&namespace, "namespace", "", "The namespace of the pipelinerun.")
	rootCmd.MarkFlagRequired("namespace")
	rootCmd.Flags().StringVar(&prName, "prName", "", "The name of the pipelinerun.")
	rootCmd.MarkFlagRequired("prName")
	rootCmd.Flags().StringVar(&taskList, "taskList", "", "The comma separated list of the tasks.")
	rootCmd.MarkFlagRequired("taskList")
	rootCmd.Flags().StringSliceVarP(&conditions, "condition", "c", []string{}, "The conditions to watch")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func sanitize_parameter_name(param string) string {
	return strings.ReplaceAll(param, "-", `\-`)
}

func sanitize_task_result(result string) interface{} {
	result = strings.TrimSpace(result)
	i, err := strconv.Atoi(result)
	if err != nil {
		return result
	}
	return i
}

func checkConditions(crs []conditionResult, tr *v1beta1.TaskRun) (string, bool) {
	for _, cr := range crs {
		parameters := make(map[string]interface{})
		for _, result := range cr.results {
			trLabel := tr.Labels["tekton.dev/pipelineTask"]
			var found bool
			for _, taskRunResults := range tr.Status.TaskRunResults {
				if result == taskRunResults.Name {
					// Do not need sanitize parameter name but only for expression for go valuate
					parameters[`results_`+trLabel+`_`+result] = sanitize_task_result(taskRunResults.Value)
					found = true
					break
				}
			}
			if !found {
				log.Printf("The result %s does not exist in taskrun %s.", result, trLabel)
				return cr.condition, false
			}
		}
		expr, err := govaluate.NewEvaluableExpression(sanitize_parameter_name(cr.condition))
		if err != nil {
			log.Fatal("syntax error:", err)
		}

		evaluateresult, err := expr.Evaluate(parameters)
		if err != nil {
			log.Fatal("evaluate error:", err)
		}

		if result, ok := evaluateresult.(bool); ok {
			if result {
				continue
			}
		}
		return cr.condition, false
	}
	return "", true
}

func watch(cmd *cobra.Command, args []string) {
	log.Printf("Starting to watch taskrun for '%s' and condition in %s/%s.", taskList, namespace, prName)

	tasks := strings.Split(taskList, ",")

	parse_conditions(conditions, &tasks)

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

	var failedTasks []string
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
				taskrunStatus := taskrun.Status.GetCondition(apis.ConditionSucceeded)
				var taskFailed bool
				if taskrunStatus.IsTrue() {
					log.Printf("The TaskRun of %s succeeded.", taskLabel)
					conditions, ok := conditionMap[taskLabel]
					if !ok { // no conditions to be passed --> any-sequencer success
						watcher.Stop()
						os.Exit(0)
					}

					condition, ok := checkConditions(conditions, taskrun)
					if ok { // condition passed -->  any-sequencer success
						watcher.Stop()
						os.Exit(0)
					}
					taskFailed = true
					log.Printf("The condition %s for the task %s does not meet.", condition, taskLabel)
				}

				if taskrunStatus.IsFalse() {
					taskFailed = true
				}

				if taskFailed && !contains(failedTasks, taskLabel) {
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
