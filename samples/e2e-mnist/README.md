# MNIST End to End example with Kubeflow components

This notebook demonstrates how to compile and execute an End to End Machine Learning workflow that uses Katib, TFJob, KFServing, and Tekton pipeline. This notebook is originated from the Kubeflow pipeline's [e2e-mnist](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/kubeflow-e2e-mnist) example that's running on Kubeflow 1.3.1. We have modified this notebook to run with Tekton support.

This pipeline contains 5 steps, it finds the best hyperparameter using Katib, creates PVC for storing models, processes the hyperparameter results, distributedly trains the model on TFJob with the best hyperparameter using more iterations, and finally serves the model using KFServing. You can visit this [medium blog](https://medium.com/@liuhgxa/an-end-to-end-use-case-by-kubeflow-b2f72b0b587) for more details on this pipeline.

To run this pipeline, make sure your cluster has at least 16 cpu and 32GB in total. Otherwise some jobs might not able to run because TFJob needs to run 4 TensorFlow pods in parallel for distributed training.

## Prerequisites
- Install [KFP Tekton prerequisites](/samples/README.md)
- Install the necessary Python packages for running Jupyter notebook.
    ```shell
    pip install jupyter numpy Pillow
    pip install kubeflow-katib==0.12.0rc0
    ```
- Make sure the Kubernetes Cluster has a [storageclass](https://kubernetes.io/docs/concepts/storage/storage-classes/) that supports ReadWriteMany in order to run distributed training.
- When running KFP single user mode, give cluster admin to the KFP service account in order to run Katib and KFServing on any namespace.
    ```shell
    kubectl create clusterrolebinding pipeline-runner-extend --clusterrole cluster-admin --serviceaccount=kubeflow:pipeline-runner
    ```
## Instructions

Once you have completed all the prerequisites for this example, then you can start the Jupyter server in this directory and click on the `mnist.ipynb` notebook. The notebook has step by step instructions for running the KFP Tekton pipeline.
```
python -m jupyter notebook
```

Or, you can compile the pipeline directly with:
```
python e2e-mnist.py
```

## Acknowledgements

Thanks [Hougang Liu](https://github.com/hougangliu) and [Andrey Velichkevich](https://github.com/andreyvelich) for creating the original e2e-mnist notebook.
