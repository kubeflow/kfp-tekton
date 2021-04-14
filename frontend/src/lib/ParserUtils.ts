/**
* Copyright 2021 Google LLC
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*      http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*/
export function parseTaskDisplayName(taskSpec?: any): string | undefined {
  const metadata = taskSpec['metadata'] || {};
  const annotations = metadata['annotations'] || false;
  if (!annotations) {
    return undefined;
  }
  const taskDisplayName = annotations['pipelines.kubeflow.org/task_display_name'];
  let componentDisplayName: string | undefined;
  try {
    componentDisplayName = JSON.parse(annotations['pipelines.kubeflow.org/component_spec'])
      .name;
  } catch (err) {
    // Expected error: metadata is missing or malformed
  }
  return taskDisplayName || componentDisplayName;
}

export function parseTaskDisplayNameByNodeId(nodeId: string, workflow?: any): string {
  let node: any;
  for (const taskRunId of Object.getOwnPropertyNames(workflow.status.taskRuns)) {
    const taskRun = workflow.status.taskRuns[taskRunId];
    if (taskRun.status && taskRun.pipelineTaskName === nodeId) {
      node = taskRun;
    }
  }

  if (!node) {
    return nodeId;
  }

  const workflowName = workflow?.metadata?.name || '';
  let displayName = node.displayName || node.id;
  if (node.name === `${workflowName}.onExit`) {
    displayName = `onExit - ${node.templateName}`;
  }
  if (workflow?.spec && workflow?.spec.pipelineSpec.tasks) {
    const tmpl = workflow.spec.pipelineSpec.tasks.find((t: any) => {console.log(t); return t?.name === nodeId});
    displayName = parseTaskDisplayName(tmpl?.taskSpec) || displayName;
  }
  return displayName;
}
