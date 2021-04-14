# Kubeflow Katib Component Samples

The sample demonstrate how to create a Kubeflow Pipeline using
[Katib](https://github.com/kubeflow/katib).
The source code for the Katib Pipeline component can be found
[here](https://github.com/kubeflow/pipelines/blob/master/components/kubeflow/katib-launcher/src/launch_experiment.py).

Check the following examples:

- Run Pipeline from Jupyter Notebook using Katib Experiment with
  [random search algorithm and early stopping](early-stopping.ipynb).

Please note that katib currently does not support k8s version 1.19.x,
it is currently being worked on at https://github.com/kubeflow/katib/pull/1450.

