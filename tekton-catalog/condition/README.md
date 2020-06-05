# condition
Conditional statements are frequently used in Kubeflow Pipelines, and as such a Tekton Catalog Task for conditions has been provided as a simple way to 
invoke conditions. This is pending the support for formal Conditions spec in Tekton V1beta1 API, which is being developed according the [design specs in Tekton community](https://docs.google.com/document/d/1kESrgmFHnirKNS4oDq3mucuB_OycBm6dSCSwRUHccZg/edit).

## Parameters
- operand1 - The left hand side operand of the condition
- operand2 - The right hand side operand of the condition
- operator - The conditional operator used to compare the two operands (e.g. == or >)
