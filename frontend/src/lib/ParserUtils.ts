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

export function parseTaskDisplayNameByNodeId(nodeId: string, workflow?: Workflow): string {
  const node = workflow?.status.nodes[nodeId];
  if (!node) {
    return nodeId;
  }
  const workflowName = workflow?.metadata?.name || '';
  let displayName = node.displayName || node.id;
  if (node.name === `${workflowName}.onExit`) {
    displayName = `onExit - ${node.templateName}`;
  }
  if (workflow?.spec && workflow?.spec.templates) {
    const tmpl = workflow.spec.templates.find(t => t?.name === node.templateName);
    displayName = parseTaskDisplayName(tmpl?.metadata) || displayName;
  }
  return displayName;
}
