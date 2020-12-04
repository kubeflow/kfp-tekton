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
