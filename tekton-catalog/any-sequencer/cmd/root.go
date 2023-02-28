/*
Copyright 2021 kubeflow.org.

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

	"github.com/spf13/cobra"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/apis/run/v1alpha1"
	customRun "github.com/tektoncd/pipeline/pkg/apis/run/v1beta1"
	tektoncdclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/apis"
)

var (
	namespace      string
	prName         string
	taskList       string
	statusPath     string
	skippingPolicy string
	errorPolicy    string
	conditions     []string
	conditionMap   map[string][]conditionResult
)

const (
	succeededStatus = "Completed"
	failedStatus    = "Failed"
	skippedStatus   = "Skipped"
	skipOnNoMatch   = "skipOnNoMatch"
	errorOnNoMatch  = "errorOnNoMatch"
	continueOnError = "continueOnError"
	failOnError     = "failOnError"
)

type conditionResult struct {
	condition string
	results   []string
}

func exitWithStatus(statusToWrite string, osStatus int) {
	if statusPath == "" {
		fmt.Printf("Program exit status is %d.\n", osStatus)
		os.Exit(osStatus)
	}
	if statusToWrite == skippedStatus {
		fmt.Println("All the tasks or conditions to watch does not meet, skipping.")
		if skippingPolicy == skipOnNoMatch {
			if err := writeStringToFile(skippedStatus, statusPath); err != nil {
				os.Exit(1)
			}
			os.Exit(0)
		}
		if skippingPolicy == errorOnNoMatch {
			writeStringToFile(skippedStatus, statusPath)
			os.Exit(1)
		}
	}
	if errorPolicy == continueOnError {
		if err := writeStringToFile(statusToWrite, statusPath); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	if errorPolicy == failOnError {
		if err := writeStringToFile(statusToWrite, statusPath); err != nil {
			os.Exit(1)
		}
		os.Exit(osStatus)
	}
	os.Exit(osStatus)
}

func writeStringToFile(s string, path string) error {
	_, err := os.Stat(path)
	var dstFile *os.File
	if err != nil {
		if os.IsNotExist(err) {
			dstFile, err = os.Create(path)
			if err != nil {
				fmt.Println("Error creating file: " + err.Error())
				return err
			}
		}
	}
	defer dstFile.Close()
	_, err = dstFile.WriteString(s)
	if err != nil {
		fmt.Println("Error writing to file: " + err.Error())
		return err
	}
	fmt.Printf("Wrote %s to file %s.\n", s, path)
	return nil
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == strings.TrimSpace(str) {
			return true
		}
	}
	return false
}

func intersection(a, b []string) (c []string) {
	m := make(map[string]bool)

	for _, item := range a {
		m[item] = true
	}

	for _, item := range b {
		if _, ok := m[item]; ok {
			c = append(c, item)
		}
	}
	return
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
			fmt.Println("The conditon must be as format 'operand1 operator operand2'.")
			exitWithStatus(failedStatus, 1)
		}
		var taskName string
		var resultName []string
		resultMatcher := regexp.MustCompile(`results_([^_]*)_([^_]+)`)
		operand1Results := resultMatcher.FindAllStringSubmatch(operands[0], -1)
		operand2Results := resultMatcher.FindAllStringSubmatch(operands[2], -1)
		if len(operand1Results) == 0 && len(operand2Results) == 0 {
			fmt.Println("Must at least contain one result and at most two in one condition for a task.")
			exitWithStatus(failedStatus, 1)
		}
		if len(operand1Results) > 0 && len(operand2Results) > 0 {
			if operand1Results[0][1] != operand2Results[0][1] {
				fmt.Println("The conditon can only contain results in one task, here's two.")
				exitWithStatus(failedStatus, 1)
			}
			taskName = operand1Results[0][1]
			resultName = []string{
				operand1Results[0][2],
				operand2Results[0][2],
			}
		} else if len(operand1Results) > 0 {
			taskName = operand1Results[0][1]
			resultName = []string{
				operand1Results[0][2],
			}
		} else if len(operand2Results) > 0 {
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
		Use:   "any-task",
		Short: "Watch taskrun or run, and exit when any taskrun or run complete",
		Long: `Watch taskrun or run, and exit when any of below is true:
			1: taskrun or run complete
			2: condition met`,
		Run: watch,
	}

	rootCmd.Flags().StringVar(&namespace, "namespace", "", "The namespace of the pipelinerun.")
	rootCmd.MarkFlagRequired("namespace")
	rootCmd.Flags().StringVar(&prName, "prName", "", "The name of the pipelinerun.")
	rootCmd.MarkFlagRequired("prName")
	rootCmd.Flags().StringVar(&taskList, "taskList", "", "The comma separated list of the tasks.")
	rootCmd.Flags().StringSliceVarP(&conditions, "condition", "c", []string{}, "The conditions to watch.")
	rootCmd.Flags().StringVar(&statusPath, "statusPath", "", "The path to write the status when finished.")
	rootCmd.Flags().StringVar(&skippingPolicy, "skippingPolicy", "", "Determines for reacting to no-dependency-condition-matching case. \"skip_on_no_match\" or \"error_on_no_match\".")
	rootCmd.Flags().StringVar(&errorPolicy, "errorPolicy", "", "An action taken when the run has failed. One of: \"fail_on_error\", \"continue_on_error\"")

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

func checkTaskrunConditions(crs []conditionResult, taskRunStatus *v1beta1.TaskRunStatus, trName string) (string, bool) {
	for _, cr := range crs {
		parameters := make(map[string]interface{})
		for _, result := range cr.results {
			var found bool
			for _, taskRunResults := range taskRunStatus.TaskRunResults {
				if result == taskRunResults.Name {
					// Do not need sanitize parameter name but only for expression for go valuate
					parameters[`results_`+trName+`_`+result] = sanitize_task_result(taskRunResults.Value.StringVal)
					found = true
					break
				}
			}
			if !found {
				fmt.Printf("The result %s does not exist in taskrun %s.\n", result, trName)
				return cr.condition, false
			}
		}
		expr, err := govaluate.NewEvaluableExpression(sanitize_parameter_name(cr.condition))
		if err != nil {
			fmt.Println("syntax error:", err)
		}

		evaluateresult, err := expr.Evaluate(parameters)
		if err != nil {
			fmt.Println("evaluate error:", err)
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

func checkRunConditions(crs []conditionResult, runStatus *customRun.CustomRunStatus, runName string) (string, bool) {
	for _, cr := range crs {
		parameters := make(map[string]interface{})
		for _, result := range cr.results {
			var found bool
			for _, runResults := range runStatus.Results {
				if result == runResults.Name {
					// Do not need sanitize parameter name but only for expression for go valuate
					parameters[`results_`+runName+`_`+result] = sanitize_task_result(runResults.Value)
					found = true
					break
				}
			}
			if !found {
				fmt.Printf("The result %s does not exist in run %s.\n", result, runName)
				return cr.condition, false
			}
		}
		expr, err := govaluate.NewEvaluableExpression(sanitize_parameter_name(cr.condition))
		if err != nil {
			fmt.Println("syntax error:", err)
		}

		evaluateresult, err := expr.Evaluate(parameters)
		if err != nil {
			fmt.Println("evaluate error:", err)
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

func checkTaskRunStatus(taskRunStatus *v1beta1.PipelineRunTaskRunStatus, failedOrSkippedTasksCh chan string) {
	var taskFailed bool
	taskName := taskRunStatus.PipelineTaskName
	taskrunStatusCondition := taskRunStatus.Status.GetCondition(apis.ConditionSucceeded)
	if taskrunStatusCondition.IsTrue() {
		fmt.Printf("The TaskRun of %s succeeded.\n", taskName)
		conditions, ok := conditionMap[taskName]
		if !ok { // no conditions to be passed --> any-sequencer success
			exitWithStatus(succeededStatus, 0)
		}

		condition, ok := checkTaskrunConditions(conditions, taskRunStatus.Status, taskName)
		if ok { // condition passed -->  any-sequencer success
			exitWithStatus(succeededStatus, 0)
		}
		taskFailed = true
		fmt.Printf("The condition %s for the task %s does not meet.\n", condition, taskName)
	}

	if taskrunStatusCondition.IsFalse() {
		taskFailed = true
	}

	if taskFailed {
		failedOrSkippedTasksCh <- taskName
	}
}

func checkRunStatus(runStatus *v1beta1.PipelineRunRunStatus, failedOrSkippedTasksCh chan string) {
	taskName := runStatus.PipelineTaskName
	var taskFailed bool
	runStatusCondition := runStatus.Status.GetCondition(apis.ConditionSucceeded)
	if runStatusCondition.IsTrue() {
		fmt.Printf("The Run of %s succeeded.\n", taskName)
		conditions, ok := conditionMap[taskName]
		if !ok { // no conditions to be passed --> any-sequencer success
			exitWithStatus(succeededStatus, 0)
		}
		condition, ok := checkRunConditions(conditions, runStatus.Status, taskName)
		if ok { // condition passed -->  any-sequencer success
			exitWithStatus(succeededStatus, 0)
		}
		taskFailed = true
		fmt.Printf("The condition %s for the task %s does not meet.\n", condition, taskName)
	}

	if runStatusCondition.IsFalse() {
		taskFailed = true
	}

	if taskFailed {
		failedOrSkippedTasksCh <- taskName
	}
}

func checkChildReferencesStatus(childReferencesStatus v1beta1.ChildStatusReference, failedOrSkippedTasksCh chan string,
	tektonClient *tektoncdclientset.Clientset) {
	taskName := childReferencesStatus.PipelineTaskName
	var taskFailed bool
	var childReferencesStatusCondition *apis.Condition
	switch childReferencesStatus.Kind {
	case "TaskRun":
		taskrun, err := tektonClient.TektonV1beta1().TaskRuns(namespace).Get(context.Background(), childReferencesStatus.Name, v1.GetOptions{})
		if err != nil {
			fmt.Println("Can't fetch taskrun: ", err)
		}
		childReferencesStatusCondition = taskrun.Status.GetCondition(apis.ConditionSucceeded)
		if childReferencesStatusCondition.IsTrue() {
			fmt.Printf("The Task of %s succeeded.\n", taskName)
			conditions, ok := conditionMap[taskName]
			if !ok { // no conditions to be passed --> any-sequencer success
				exitWithStatus(succeededStatus, 0)
			}
			condition, ok := checkTaskrunConditions(conditions, &taskrun.Status, taskName)
			if ok { // condition passed -->  any-sequencer success
				exitWithStatus(succeededStatus, 0)
			}
			taskFailed = true
			fmt.Printf("The condition %s for the task %s does not meet.\n", condition, taskName)
		}
	case "Run":
		run, err := tektonClient.TektonV1alpha1().Runs(namespace).Get(context.Background(), childReferencesStatus.Name, v1.GetOptions{})
		if err != nil {
			fmt.Println("Can't fetch run: ", err)
		}
		childReferencesStatusCondition = run.Status.GetCondition(apis.ConditionSucceeded)
		if childReferencesStatusCondition.IsTrue() {
			fmt.Printf("The Task of %s succeeded.\n", taskName)
			conditions, ok := conditionMap[taskName]
			if !ok { // no conditions to be passed --> any-sequencer success
				exitWithStatus(succeededStatus, 0)
			}
			condition, ok := checkRunConditions(conditions, FromRunStatus(&run.Status), taskName)
			if ok { // condition passed -->  any-sequencer success
				exitWithStatus(succeededStatus, 0)
			}
			taskFailed = true
			fmt.Printf("The condition %s for the task %s does not meet.\n", condition, taskName)
		}
	case "CustomRun":
		customRun, err := tektonClient.TektonV1beta1().CustomRuns(namespace).Get(context.Background(), childReferencesStatus.Name, v1.GetOptions{})
		if err != nil {
			fmt.Println("Can't fetch customrun: ", err)
		}
		childReferencesStatusCondition = customRun.Status.GetCondition(apis.ConditionSucceeded)
		if childReferencesStatusCondition.IsTrue() {
			fmt.Printf("The Task of %s succeeded.\n", taskName)
			conditions, ok := conditionMap[taskName]
			if !ok { // no conditions to be passed --> any-sequencer success
				exitWithStatus(succeededStatus, 0)
			}
			condition, ok := checkRunConditions(conditions, &customRun.Status, taskName)
			if ok { // condition passed -->  any-sequencer success
				exitWithStatus(succeededStatus, 0)
			}
			taskFailed = true
			fmt.Printf("The condition %s for the task %s does not meet.\n", condition, taskName)
		}
	default:
	}

	if childReferencesStatusCondition.IsFalse() {
		taskFailed = true
	}

	if taskFailed {
		failedOrSkippedTasksCh <- taskName
	}
}

func checkSkippedTasks(pr *v1beta1.PipelineRun, tasks []string, failedOrSkippedTasksCh chan string) {
	var skippedTasks []string

	if pr.Status.SkippedTasks != nil {
		for _, skippedTask := range pr.Status.SkippedTasks {
			if !contains(skippedTasks, skippedTask.Name) {
				skippedTasks = append(skippedTasks, skippedTask.Name)
			}
		}
		interestedSkippedTasks := intersection(skippedTasks, tasks)
		for _, interestedSkippedTask := range interestedSkippedTasks {
			failedOrSkippedTasksCh <- interestedSkippedTask
		}
	}
}

func watchPipelineRun(tasks []string, failedOrSkippedTasksCh chan string) []string {

	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Printf("Get config of the cluster failed: %+v \n", err)
		exitWithStatus(failedStatus, 1)
	}
	tektonClient, err := tektoncdclientset.NewForConfig(config)
	if err != nil {
		fmt.Printf("Get client of tekton failed: %+v \n", err)
		exitWithStatus(failedStatus, 1)
	}

	fieldSelector := "metadata.name=" + strings.TrimSpace(prName)

	for {
		prWatcher, err := tektonClient.TektonV1beta1().PipelineRuns(namespace).Watch(context.TODO(), metav1.ListOptions{FieldSelector: fieldSelector})

		if err != nil {
			fmt.Println("Run Watcher error:" + err.Error())
			fmt.Println("Please ensure the service account has permission to get PipelineRun.")
			exitWithStatus(failedStatus, 1)
		}

		for event := range prWatcher.ResultChan() {
			pr := event.Object.(*v1beta1.PipelineRun)
			for _, childReferencesStatus := range pr.Status.ChildReferences {
				if contains(tasks, childReferencesStatus.PipelineTaskName) {
					checkChildReferencesStatus(childReferencesStatus, failedOrSkippedTasksCh, tektonClient)
				}
			}
			for _, taskRunStatus := range pr.Status.TaskRuns {
				if contains(tasks, taskRunStatus.PipelineTaskName) {
					checkTaskRunStatus(taskRunStatus, failedOrSkippedTasksCh)
				}
			}
			for _, runStatus := range pr.Status.Runs {
				if contains(tasks, runStatus.PipelineTaskName) {
					checkRunStatus(runStatus, failedOrSkippedTasksCh)
				}
			}
			checkSkippedTasks(pr, tasks, failedOrSkippedTasksCh)
		}
	}
}

func watch(cmd *cobra.Command, args []string) {
	if taskList == "" && len(conditions) == 0 {
		fmt.Println("Should provide either taskList or conditions to watch.")
		exitWithStatus(failedStatus, 1)
	}

	if statusPath != "" {
		if skippingPolicy == "" {
			skippingPolicy = skipOnNoMatch
		} else {
			if skippingPolicy != skipOnNoMatch && skippingPolicy != errorOnNoMatch {
				fmt.Printf("skippingPolicy value must be one of %s or %s.\n", skipOnNoMatch, errorOnNoMatch)
				exitWithStatus(failedStatus, 1)
			}
		}
		if errorPolicy == "" {
			errorPolicy = continueOnError
		} else {
			if errorPolicy != continueOnError && errorPolicy != failOnError {
				fmt.Printf("skippingPolicy value must be one of %s or %s.\n", continueOnError, failOnError)
				exitWithStatus(failedStatus, 1)
			}
		}
	}

	var tasks []string
	if taskList != "" {
		tasks = strings.Split(taskList, ",")
	}

	parse_conditions(conditions, &tasks)

	fmt.Printf("Starting to watch taskrun or run of '%s' and conditions in %s/%s.\n", taskList, namespace, prName)

	failedOrSkippedTasksCh := make(chan string)

	go watchPipelineRun(tasks, failedOrSkippedTasksCh)

	var failedOrSkippedTasks []string
	for failedorSkippedTask := range failedOrSkippedTasksCh {
		if !contains(failedOrSkippedTasks, failedorSkippedTask) {
			fmt.Printf("The taskrun or run failed/skipped, or condition unmatched of task: %s.\n", failedorSkippedTask)
			failedOrSkippedTasks = append(failedOrSkippedTasks, failedorSkippedTask)
		}
		if len(failedOrSkippedTasks) >= len(tasks) {
			fmt.Println("All specified TaskRun(s) or Run(s) failed or skipped.")
			exitWithStatus(skippedStatus, 1)
		}
	}
}

// FromRunStatus converts a *v1alpha1.RunStatus into a corresponding *v1beta1.CustomRunStatus
func FromRunStatus(orig *v1alpha1.RunStatus) *customRun.CustomRunStatus {
	crs := customRun.CustomRunStatus{
		Status: orig.Status,
		CustomRunStatusFields: customRun.CustomRunStatusFields{
			StartTime:      orig.StartTime,
			CompletionTime: orig.CompletionTime,
			ExtraFields:    orig.ExtraFields,
		},
	}

	for _, origRes := range orig.Results {
		crs.Results = append(crs.Results, customRun.CustomRunResult{
			Name:  origRes.Name,
			Value: origRes.Value,
		})
	}

	for _, origRetryStatus := range orig.RetriesStatus {
		crs.RetriesStatus = append(crs.RetriesStatus, customRun.FromRunStatus(origRetryStatus))
	}

	return &crs
}
