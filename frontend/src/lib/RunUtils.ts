/*
 * Copyright 2018 The Kubeflow Authors
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

import { V1Parameter, V1PipelineVersion } from '../apis/pipeline';
import { Workflow } from '../third_party/mlmd/argo_template';
import { V1Job } from '../apis/job';
import {
  V1PipelineRuntime,
  V1ResourceReference,
  V1ResourceType,
  V1Run,
  V1RunDetail,
  V1Relationship,
} from '../apis/run';
import { logger } from './Utils';
import WorkflowParser from './WorkflowParser';
import { V1Experiment } from 'src/apis/experiment';

export interface MetricMetadata {
  count: number;
  maxValue: number;
  minValue: number;
  name: string;
}

export interface ExperimentInfo {
  displayName?: string;
  id: string;
}

function getParametersFromRun(run: V1RunDetail): V1Parameter[] {
  return getParametersFromRuntime(run.pipeline_runtime);
}

function getParametersFromRuntime(runtime?: V1PipelineRuntime): V1Parameter[] {
  if (!runtime) {
    return [];
  }

  try {
    const workflow = JSON.parse(runtime.workflow_manifest!) as Workflow;
    return WorkflowParser.getParameters(workflow);
  } catch (err) {
    logger.error('Failed to parse runtime workflow manifest', err);
    return [];
  }
}

function getPipelineId(run?: V1Run | V1Job): string | null {
  return (run && run.pipeline_spec && run.pipeline_spec.pipeline_id) || null;
}

function getPipelineName(run?: V1Run | V1Job): string | null {
  return (run && run.pipeline_spec && run.pipeline_spec.pipeline_name) || null;
}

function getPipelineVersionId(run?: V1Run | V1Job): string | null {
  return run &&
    run.resource_references &&
    run.resource_references.some(
      ref => ref.key && ref.key.type && ref.key.type === V1ResourceType.PIPELINEVERSION,
    )
    ? run.resource_references.find(
        ref => ref.key && ref.key.type && ref.key.type === V1ResourceType.PIPELINEVERSION,
      )!.key!.id!
    : null;
}

function getPipelineIdFromApiPipelineVersion(
  pipelineVersion?: V1PipelineVersion,
): string | undefined {
  return pipelineVersion &&
    pipelineVersion.resource_references &&
    pipelineVersion.resource_references.some(
      ref => ref.key && ref.key.type && ref.key.id && ref.key.type === V1ResourceType.PIPELINE,
    )
    ? pipelineVersion.resource_references.find(
        ref => ref.key && ref.key.type && ref.key.id && ref.key.type === V1ResourceType.PIPELINE,
      )!.key!.id!
    : undefined;
}

function getWorkflowManifest(run?: V1Run | V1Job): string | null {
  return (run && run.pipeline_spec && run.pipeline_spec.workflow_manifest) || null;
}

function getFirstExperimentReference(run?: V1Run | V1Job): V1ResourceReference | null {
  return getAllExperimentReferences(run)[0] || null;
}

function getFirstExperimentReferenceId(run?: V1Run | V1Job): string | null {
  const reference = getFirstExperimentReference(run);
  return (reference && reference.key && reference.key.id) || null;
}

function getFirstExperimentReferenceName(run?: V1Run | V1Job): string | null {
  const reference = getFirstExperimentReference(run);
  return (reference && reference.name) || null;
}

function getAllExperimentReferences(run?: V1Run | V1Job): V1ResourceReference[] {
  return ((run && run.resource_references) || []).filter(
    ref => (ref.key && ref.key.type && ref.key.type === V1ResourceType.EXPERIMENT) || false,
  );
}

function getNamespaceReferenceName(run?: V1Experiment): string | undefined {
  // There should be only one namespace reference.
  const namespaceRef =
    run &&
    run.resource_references &&
    run.resource_references.find(
      ref =>
        ref.relationship === V1Relationship.OWNER &&
        ref.key &&
        ref.key.type === V1ResourceType.NAMESPACE,
    );
  return namespaceRef && namespaceRef.key && namespaceRef.key.id;
}

/**
 * Takes an array of Runs and returns a map where each key represents a single metric, and its value
 * contains the name again, how many of that metric were collected across all supplied Runs, and the
 * max and min values encountered for that metric.
 */
function runsToMetricMetadataMap(runs: V1Run[]): Map<string, MetricMetadata> {
  return runs.reduce((metricMetadatas, run) => {
    if (!run || !run.metrics) {
      return metricMetadatas;
    }
    run.metrics.forEach(metric => {
      if (!metric.name || metric.number_value === undefined || isNaN(metric.number_value)) {
        return;
      }

      let metricMetadata = metricMetadatas.get(metric.name);
      if (!metricMetadata) {
        metricMetadata = {
          count: 0,
          maxValue: Number.MIN_VALUE,
          minValue: Number.MAX_VALUE,
          name: metric.name,
        };
        metricMetadatas.set(metricMetadata.name, metricMetadata);
      }
      metricMetadata.count++;
      metricMetadata.minValue = Math.min(metricMetadata.minValue, metric.number_value);
      metricMetadata.maxValue = Math.max(metricMetadata.maxValue, metric.number_value);
    });
    return metricMetadatas;
  }, new Map<string, MetricMetadata>());
}

function extractMetricMetadata(runs: V1Run[]): MetricMetadata[] {
  return Array.from(runsToMetricMetadataMap(runs).values());
}

function getRecurringRunId(run?: V1Run): string {
  if (!run) {
    return '';
  }

  for (const ref of run.resource_references || []) {
    if (ref.key && ref.key.type === V1ResourceType.JOB) {
      return ref.key.id || '';
    }
  }
  return '';
}

function getRecurringRunName(run?: V1Run): string {
  if (!run) {
    return '';
  }

  for (const ref of run.resource_references || []) {
    if (ref.key && ref.key.type === V1ResourceType.JOB) {
      return ref.name || '';
    }
  }
  return '';
}

// TODO: This file needs tests
const RunUtils = {
  extractMetricMetadata,
  getAllExperimentReferences,
  getFirstExperimentReference,
  getFirstExperimentReferenceId,
  getFirstExperimentReferenceName,
  getNamespaceReferenceName,
  getParametersFromRun,
  getParametersFromRuntime,
  getPipelineId,
  getPipelineIdFromApiPipelineVersion,
  getPipelineName,
  getPipelineVersionId,
  getRecurringRunId,
  getRecurringRunName,
  getWorkflowManifest,
  runsToMetricMetadataMap,
};
export default RunUtils;
