# Kubeflow Katib Component Samples

These samples demonstrate how to create a Kubeflow Pipeline using
[Katib](https://github.com/kubeflow/katib).
The source code for the Katib Pipeline component can be found
[here](https://github.com/kubeflow/pipelines/blob/master/components/kubeflow/katib-launcher/src/launch_experiment.py).

Check the following examples:

- Run Pipeline from Jupyter Notebook using Katib Experiment with
  [random search algorithm and early stopping](early-stopping.ipynb).

- Compile compressed YAML definition of the Pipeline using Katib Experiment with
  [Kubeflow MPIJob and Horovod training container](mpi-job-horovod.py).

Please note that katib currently does not support k8s version 1.19.x,
it currently being worked on at https://github.com/kubeflow/katib/pull/1450.

Note: We need to install [MPIjob controller](https://github.com/kubeflow/mpi-operator), for this example to work.