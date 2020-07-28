/*
 * Copyright 2018-2019 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import * as dagre from 'dagre';
import {
  NodeStatus,
  Parameter,
  S3Artifact,
  Workflow,
} from '../../third_party/argo-ui/argo_template';
import { statusToIcon } from '../pages/Status';
import { Constants } from './Constants';
import { KeyValue } from './StaticGraphParser';
import { NodePhase, statusToBgColor, statusToPhase } from './StatusUtils';
import { isS3Endpoint } from './AwsHelper';

export enum StorageService {
  GCS = 'gcs',
  HTTP = 'http',
  HTTPS = 'https',
  MINIO = 'minio',
  S3 = 's3',
}

export interface StoragePath {
  source: StorageService;
  bucket: string;
  key: string;
}

export default class WorkflowParser {
  public static createRuntimeGraph(workflow: any): dagre.graphlib.Graph {
    const graph = new dagre.graphlib.Graph();
    graph.setGraph({});
    graph.setDefaultEdgeLabel(() => ({}));

    // If a run exists but has no status is available yet return an empty graph
    if (
      workflow &&
      workflow.status &&
      (Object.keys(workflow.status).length === 0 || !workflow.status.taskRuns)
    )
      return graph;

    const tasks = (workflow['spec']['pipelineSpec']['tasks'] || []).concat(
      workflow['spec']['pipelineSpec']['finally'] || [],
    );
    const status = workflow['status']['taskRuns'];
    const pipelineParams = workflow['spec']['params'];
    const exitHandlers =
      (workflow['spec']['pipelineSpec']['finally'] || []).map((element: any) => {
        return element['name'];
      }) || [];

    // Create a map from task name to status for every status received
    const statusMap = new Map<string, any>();
    for (const taskRunId of Object.getOwnPropertyNames(status)) {
      status[taskRunId]['taskRunId'] = taskRunId;
      if (status[taskRunId]['status'])
        statusMap.set(status[taskRunId]['pipelineTaskName'], status[taskRunId]);
    }

    for (const task of tasks) {
      // If the task has a status then add it and its edges to the graph
      if (statusMap.get(task['name'])) {
        const conditions = task['conditions'] || [];
        const taskId =
          statusMap.get(task['name']) && statusMap.get(task['name'])!['status']['podName'] !== ''
            ? statusMap.get(task['name'])!['status']['podName']
            : task['name'];
        const edges = this.checkParams(statusMap, pipelineParams, task, '');

        // Add all of this Task's conditional dependencies as Task dependencies
        for (const condition of conditions)
          edges.push(...this.checkParams(statusMap, pipelineParams, condition, taskId));

        if (task['runAfter']) {
          task['runAfter'].forEach((parentTask: any) => {
            if (
              statusMap.get(parentTask) &&
              statusMap.get(parentTask)!['status']['conditions'][0]['type'] === 'Succeeded'
            ) {
              const parentId = statusMap.get(parentTask)!['status']['podName'];
              edges.push({ parent: parentId, child: taskId });
            }
          });
        }

        for (const edge of edges || []) graph.setEdge(edge['parent'], edge['child']);

        const status = this.getStatus(statusMap.get(task['name']));
        const phase = statusToPhase(status);
        const statusColoring = exitHandlers.includes(task['name'])
          ? '#fef7f0'
          : statusToBgColor(phase, '');
        // Add a node for the Task
        graph.setNode(taskId, {
          height: Constants.NODE_HEIGHT,
          icon: statusToIcon(status),
          label: task['name'],
          statusColoring: statusColoring,
          width: Constants.NODE_WIDTH,
        });
      }
    }

    return graph;
  }

  private static checkParams(
    statusMap: Map<string, any>,
    pipelineParams: any,
    component: any,
    ownerTask: string,
  ): { parent: string; child: string }[] {
    const edges: { parent: string; child: string }[] = [];
    const componentId =
      ownerTask !== ''
        ? component['conditionRef']
        : statusMap.get(component['name']) &&
          statusMap.get(component['name'])!['status']['podName'] !== ''
        ? statusMap.get(component['name'])!['status']['podName']
        : component['name'];

    // Adds dependencies from task params
    for (const param of component['params'] || []) {
      let paramValue = param['value'] || '';

      // If the parameters are passed from the pipeline parameters then grab the value from the pipeline parameters
      if (
        param['value'].substring(0, 9) === '$(params.' &&
        param['value'].substring(param['value'].length - 1) === ')'
      ) {
        const paramName = param['value'].substring(9, param['value'].length - 1);
        for (const pipelineParam of pipelineParams)
          if (pipelineParam['name'] === paramName) paramValue = pipelineParam['value'];
      }
      // If the parameters are passed from the parent task's results and the task is completed then grab the resulting values
      else if (
        param['value'].substring(0, 2) === '$(' &&
        param['value'].substring(param['value'].length - 1) === ')'
      ) {
        const paramSplit = param['value'].split('.');
        const parentTask = paramSplit[1];
        const paramName = paramSplit[paramSplit.length - 1].substring(
          0,
          paramSplit[paramSplit.length - 1].length - 1,
        );

        if (
          statusMap.get(parentTask) &&
          statusMap.get(parentTask)!['status']['conditions'][0]['type'] === 'Succeeded'
        ) {
          const parentId = statusMap.get(parentTask)!['status']['podName'];
          edges.push({ parent: parentId, child: ownerTask === '' ? componentId : ownerTask });

          // Add the taskResults value to the params value in status
          for (const result of statusMap.get(parentTask)!['status']['taskResults'] || []) {
            if (result['name'] === paramName) paramValue = result['value'];
          }
        }
      }
      // Find the output that matches this input and pull the value
      if (
        statusMap.get(component['name']) &&
        statusMap.get(component['name'])['status']['taskSpec']
      ) {
        for (const statusParam of statusMap.get(component['name'])!['status']['taskSpec']['params'])
          if (statusParam['name'] === param['name']) statusParam['value'] = paramValue;
      }
    }

    return edges;
  }

  public static getStatus(execStatus: any): NodePhase {
    return execStatus!.status.conditions[0].reason;
  }

  public static getParameters(workflow?: any): Parameter[] {
    if (workflow && workflow.spec && workflow.spec.params) {
      return workflow.spec.params || [];
    }

    return [];
  }

  // Makes sure the workflow object contains the node and returns its
  // inputs/outputs if any, while looking out for any missing link in the chain to
  // the node's inputs/outputs.
  public static getNodeInputOutputParams(
    workflow?: any,
    nodeId?: string,
  ): Record<'inputParams' | 'outputParams', Array<KeyValue<string>>> {
    type ParamList = Array<KeyValue<string>>;
    let inputParams: ParamList = [];
    let outputParams: ParamList = [];
    if (!nodeId || !workflow || !workflow.status || !workflow.status.taskRuns) {
      return { inputParams, outputParams };
    }

    for (const taskRunId of Object.getOwnPropertyNames(workflow.status.taskRuns)) {
      const taskRun = workflow.status.taskRuns[taskRunId];
      if (taskRun.status && taskRun.status.podName === nodeId) {
        inputParams = taskRun.status.taskSpec.params
          ? taskRun.status.taskSpec.params.map(({ name, value }: any) => [name, value])
          : inputParams;
        outputParams = taskRun.status.taskResults
          ? taskRun.status.taskResults.map(({ name, value }: any) => [name, value])
          : outputParams;
      }
    }

    return { inputParams, outputParams };
  }

  // Makes sure the workflow object contains the node and returns its
  // inputs/outputs artifacts if any, while looking out for any missing link in the chain to
  // the node's inputs/outputs.
  public static getNodeInputOutputArtifacts(
    workflow?: Workflow,
    nodeId?: string,
  ): Record<'inputArtifacts' | 'outputArtifacts', Array<KeyValue<S3Artifact>>> {
    type ParamList = Array<KeyValue<S3Artifact>>;
    let inputArtifacts: ParamList = [];
    let outputArtifacts: ParamList = [];
    if (
      !nodeId ||
      !workflow ||
      !workflow.status ||
      !workflow.status.nodes ||
      !workflow.status.nodes[nodeId]
    ) {
      return { inputArtifacts, outputArtifacts };
    }

    const { inputs, outputs } = workflow.status.nodes[nodeId];
    if (!!inputs && !!inputs.artifacts) {
      inputArtifacts = inputs.artifacts.map(({ name, s3 }) => [name, s3]);
    }
    if (!!outputs && !!outputs.artifacts) {
      outputArtifacts = outputs.artifacts.map(({ name, s3 }) => [name, s3]);
    }
    return { inputArtifacts, outputArtifacts };
  }

  // Makes sure the workflow object contains the node and returns its
  // volume mounts if any.
  public static getNodeVolumeMounts(workflow: any, nodeId: string): Array<KeyValue<string>> {
    if (!workflow || !workflow.status || !workflow.status.taskRuns) {
      return [];
    }

    // If the matching taskRun for nodeId can be found then return the volumes found in the main step
    for (const task of Object.getOwnPropertyNames(workflow.status.taskRuns)) {
      const taskRun = workflow.status.taskRuns[task];
      if (taskRun.status && taskRun.status.podName === nodeId) {
        const steps = workflow.status.taskRuns[task].status.taskSpec.steps;
        for (const step of steps || [])
          if (step.name === 'main')
            return (step.volumeMounts || []).map((volume: any) => [volume.mountPath, volume.name]);
      }
    }
    return [];
  }

  // Makes sure the workflow object contains the node and returns its
  // action and manifest.
  public static getNodeManifest(workflow: Workflow, nodeId: string): Array<KeyValue<string>> {
    if (
      !workflow ||
      !workflow.status ||
      !workflow.status.nodes ||
      !workflow.status.nodes[nodeId] ||
      !workflow.spec ||
      !workflow.spec.templates
    ) {
      return [];
    }

    const node = workflow.status.nodes[nodeId];
    const tmpl = workflow.spec.templates.find(t => !!t && !!t.name && t.name === node.templateName);
    let manifest: Array<KeyValue<string>> = [];
    if (tmpl && tmpl.resource && tmpl.resource.action && tmpl.resource.manifest) {
      manifest = [[tmpl.resource.action, tmpl.resource.manifest]];
    }
    return manifest;
  }

  // Returns a list of output paths for the given workflow Node, by looking for
  // and the Argo artifacts syntax in the outputs section.
  public static loadNodeOutputPaths(selectedWorkflowNode: NodeStatus): StoragePath[] {
    const outputPaths: StoragePath[] = [];
    if (selectedWorkflowNode && selectedWorkflowNode.outputs) {
      (selectedWorkflowNode.outputs.artifacts || [])
        .filter(a => a.name === 'mlpipeline-ui-metadata' && !!a.s3)
        .forEach(a =>
          outputPaths.push({
            bucket: a.s3!.bucket,
            key: a.s3!.key,
            source: isS3Endpoint(a.s3!.endpoint) ? StorageService.S3 : StorageService.MINIO,
          }),
        );
    }
    return outputPaths;
  }

  // Returns a list of output paths for the entire workflow, by searching all nodes in
  // the workflow, and parsing outputs for each.
  public static loadAllOutputPaths(workflow: Workflow): StoragePath[] {
    return this.loadAllOutputPathsWithStepNames(workflow).map(entry => entry.path);
  }

  // Returns a list of object mapping a step name to output path for the entire workflow,
  // by searching all nodes in the workflow, and parsing outputs for each.
  public static loadAllOutputPathsWithStepNames(
    workflow: Workflow,
  ): Array<{ stepName: string; path: StoragePath }> {
    const outputPaths: Array<{ stepName: string; path: StoragePath }> = [];
    if (workflow && workflow.status && workflow.status.nodes) {
      Object.keys(workflow.status.nodes).forEach(n =>
        this.loadNodeOutputPaths(workflow.status.nodes[n]).map(path =>
          outputPaths.push({ stepName: workflow.status.nodes[n].displayName, path }),
        ),
      );
    }

    return outputPaths;
  }

  // Given a storage path, returns a structured object that contains the storage
  // service (currently only GCS), and bucket and key in that service.
  public static parseStoragePath(strPath: string): StoragePath {
    if (strPath.startsWith('gs://')) {
      const pathParts = strPath.substr('gs://'.length).split('/');
      return {
        bucket: pathParts[0],
        key: pathParts.slice(1).join('/'),
        source: StorageService.GCS,
      };
    } else if (strPath.startsWith('minio://')) {
      const pathParts = strPath.substr('minio://'.length).split('/');
      return {
        bucket: pathParts[0],
        key: pathParts.slice(1).join('/'),
        source: StorageService.MINIO,
      };
    } else if (strPath.startsWith('s3://')) {
      const pathParts = strPath.substr('s3://'.length).split('/');
      return {
        bucket: pathParts[0],
        key: pathParts.slice(1).join('/'),
        source: StorageService.S3,
      };
    } else if (strPath.startsWith('http://')) {
      const pathParts = strPath.substr('http://'.length).split('/');
      return {
        bucket: pathParts[0],
        key: pathParts.slice(1).join('/'),
        source: StorageService.HTTP,
      };
    } else if (strPath.startsWith('https://')) {
      const pathParts = strPath.substr('https://'.length).split('/');
      return {
        bucket: pathParts[0],
        key: pathParts.slice(1).join('/'),
        source: StorageService.HTTPS,
      };
    } else {
      throw new Error('Unsupported storage path: ' + strPath);
    }
  }

  // Outbound nodes are roughly those nodes which are the final step of the
  // workflow's execution. More information can be found in the NodeStatus
  // interface definition.
  public static getOutboundNodes(graph: Workflow, nodeId: string): string[] {
    let outbound: string[] = [];

    if (!graph || !graph.status || !graph.status.nodes) {
      return outbound;
    }

    const node = graph.status.nodes[nodeId];
    if (!node) {
      return outbound;
    }

    if (node.type === 'Pod') {
      return [node.id];
    }
    for (const outboundNodeID of node.outboundNodes || []) {
      const outNode = graph.status.nodes[outboundNodeID];
      if (outNode && outNode.type === 'Pod') {
        outbound.push(outboundNodeID);
      } else {
        outbound = outbound.concat(this.getOutboundNodes(graph, outboundNodeID));
      }
    }
    return outbound;
  }

  // Returns whether or not the given node is one of the intermediate nodes used
  // by Argo to orchestrate the workflow. Such nodes are not generally
  // meaningful from a user's perspective.
  public static isVirtual(node: NodeStatus): boolean {
    return (
      (node.type === 'StepGroup' || node.type === 'DAG' || node.type === 'TaskGroup') &&
      !!node.boundaryID
    );
  }

  // Returns a workflow-level error string if found, empty string if none
  public static getWorkflowError(workflow: Workflow): string {
    if (
      workflow &&
      workflow.status &&
      workflow.status.message &&
      (workflow.status.phase === NodePhase.ERROR || workflow.status.phase === NodePhase.FAILED)
    ) {
      return workflow.status.message;
    } else {
      return '';
    }
  }
}
