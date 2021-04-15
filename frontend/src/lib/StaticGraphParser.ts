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
import { color } from '../Css';
import { Constants } from './Constants';
import { parseTaskDisplayName } from './ParserUtils';
import { graphlib } from 'dagre';

export type nodeType = 'container' | 'resource' | 'dag' | 'unknown';

export interface KeyValue<T> extends Array<any> {
  0?: string;
  1?: T;
}

export class SelectedNodeInfo {
  public args: string[];
  public command: string[];
  public condition: string;
  public image: string;
  public inputs: Array<KeyValue<string>>;
  public nodeType: nodeType;
  public outputs: Array<KeyValue<string>>;
  public volumeMounts: Array<KeyValue<string>>;
  public resource: Array<KeyValue<string>>;

  constructor() {
    this.args = [];
    this.command = [];
    this.condition = '';
    this.image = '';
    this.inputs = [[]];
    this.nodeType = 'unknown';
    this.outputs = [[]];
    this.volumeMounts = [[]];
    this.resource = [[]];
  }
}

export function _populateInfoFromTask(info: SelectedNodeInfo, task?: any): SelectedNodeInfo {
  if (!task) {
    return info;
  }

  info.nodeType = 'container';
  if (task['taskSpec'] && task['taskSpec']['steps']) {
    const steps = task['taskSpec']['steps'];
    info.args = steps[0]['args'] || [];
    info.command = steps[0]['command'] || [];
    info.image = steps[0]['image'] || [];
    info.volumeMounts = (steps[0]['volumeMounts'] || []).map((volume: any) => [
      volume.mountPath,
      volume.name,
    ]);
  }

  if (task['taskSpec'] && task['taskSpec']['params'])
    info.inputs = (task['taskSpec']['params'] || []).map((p: any) => [p['name'], p['value'] || '']);
  if (task['taskSpec'] && task['taskSpec']['results'])
    info.outputs = (task['taskSpec']['results'] || []).map((p: any) => {
      return [p['name'], p['description'] || ''];
    });
  if (task['params']) {
    info.inputs = (task['params'] || []).map((p: any) => [p['name'], p['value'] || '']);
  }

  return info;
}

export function createGraph(workflow: any): dagre.graphlib.Graph {
  const graph = new dagre.graphlib.Graph();
  graph.setGraph({});
  graph.setDefaultEdgeLabel(() => ({}));

  buildTektonDag(graph, workflow);
  return graph;
}

function buildTektonDag(graph: dagre.graphlib.Graph, template: any): void {
  const pipeline = template;

  if (!template || !template.spec) {
    throw new Error("Graph template or template spec doesn't exist.");
  }

  const tasks = (pipeline['spec']['pipelineSpec']['tasks'] || []).concat(
    pipeline['spec']['pipelineSpec']['finally'] || [],
  );

  const exitHandlers =
    (pipeline['spec']['pipelineSpec']['finally'] || []).map((element: any) => {
      return element['name'];
    }) || [];
  // Collect the anyConditions from 'metadata.annotations.anyConditions'
  let anyConditions = {};
  if (
    pipeline['metadata'] &&
    pipeline['metadata']['annotations'] &&
    pipeline['metadata']['annotations']['anyConditions']
  ) {
    anyConditions = JSON.parse(pipeline['metadata']['annotations']['anyConditions']);
  }
  const anyTasks = Object.keys(anyConditions);
  for (const task of tasks) {
    const taskName = task['name'];

    // Checks for dependencies mentioned in the runAfter section of a task and then checks for dependencies based
    // on task output being passed in as parameters
    if (task['runAfter'])
      task['runAfter'].forEach((depTask: any) => {
        graph.setEdge(depTask, taskName);
      });
    // Adds dependencies for anySequencers from 'anyCondition' annotation
    if (anyTasks.includes(task['name'])) {
      for (const depTask of anyConditions[task['name']]) {
        graph.setEdge(depTask, taskName);
      }
    }
    // Adds any dependencies that arise from Conditions and tracks these dependencies to make sure they aren't duplicated in the case that
    // the Condition and the base task use output from the same dependency
    for (const condition of task['when'] || []) {
      const input = condition['input'];
      if (input.substring(0, 8) === '$(tasks.' && input.substring(input.length - 1) === ')') {
        const paramSplit = input.split('.');
        const parentTask = paramSplit[1];

        graph.setEdge(parentTask, taskName);
      }
    }

    // Adds any dependencies that arise from Conditions and tracks these dependencies to make sure they aren't duplicated in the case that
    // the Condition and the base task use output from the same dependency
    for (const condition of task['conditions'] || []) {
      for (const condParam of condition['params'] || []) {
        if (
          condParam['value'].substring(0, 8) === '$(tasks.' &&
          condParam['value'].substring(condParam['value'].length - 1) === ')'
        ) {
          const paramSplit = condParam['value'].split('.');
          const parentTask = paramSplit[1];

          graph.setEdge(parentTask, taskName);
        }
      }
    }

    for (const param of task['params'] || []) {
      if (param['value'].indexOf('$(tasks.') !== -1 && param['value'].indexOf(')') !== -1) {
        const paramSplit = param['value'].split('.');
        const parentTask = paramSplit[1];
        graph.setEdge(parentTask, taskName);
      }
    }

    // Add the info for this node
    const info = new SelectedNodeInfo();
    _populateInfoFromTask(info, task);

    const label = exitHandlers.includes(task['name']) ? 'onExit - ' + taskName : taskName;
    const bgColor = exitHandlers.includes(task['name'])
      ? color.lightGrey
      : task.when
      ? 'cornsilk'
      : undefined;

    graph.setNode(taskName, {
      bgColor: bgColor,
      height: Constants.NODE_HEIGHT,
      info,
      label: parseTaskDisplayName(task['taskSpec'] || task['taskRef']) || label,
      width: Constants.NODE_WIDTH,
    });
  }
}

/**
 * Perform a transitive reduction over the input graph.
 *
 * From [1]: Transitive reduction of a directed graph D is another directed
 * graph with the same vertices and as few edges as possible, such that for all
 * pairs of vertices v, w a (directed) path from v to w in D exists if and only
 * if such a path exists in the reduction
 *
 * The current implementation has a time complexity bound `O(n*m)`, where `n`
 * are the nodes and `m` are the edges of the input graph.
 *
 * [1]: https://en.wikipedia.org/wiki/Transitive_reduction
 *
 * @param graph The dagre graph object
 */
export function transitiveReduction(graph: dagre.graphlib.Graph): dagre.graphlib.Graph | undefined {
  // safeguard against too big graphs
  if (!graph || graph.edgeCount() > 1000 || graph.nodeCount() > 1000) {
    return undefined;
  }

  const result = graphlib.json.read(graphlib.json.write(graph));
  let visited: string[] = [];
  const dfs_with_removal = (current: string, parent: string) => {
    result.successors(current)?.forEach((node: any) => {
      if (visited.includes(node)) return;
      visited.push(node);
      if (result.successors(parent)?.includes(node)) {
        result.removeEdge(parent, node);
      }
      dfs_with_removal(node, parent);
    });
  };

  result.nodes().forEach(node => {
    visited = []; // clean this up before each new DFS
    // start a DFS from each successor of `node`
    result.successors(node)?.forEach((successor: any) => dfs_with_removal(successor, node));
  });
  return result;
}

export function compareGraphEdges(graph1: dagre.graphlib.Graph, graph2: dagre.graphlib.Graph) {
  return (
    graph1
      .edges()
      .map(e => `${e.name}${e.v}${e.w}`)
      .sort()
      .toString() ===
    graph2
      .edges()
      .map(e => `${e.name}${e.v}${e.w}`)
      .sort()
      .toString()
  );
}
