# MNIST End to End example with Kubeflow compoenents

This notebook demonstrates how to compile and execute an End to End Machine Learning workflow that uses Katib, TFJob, KFServing, and Tekton pipeline. This notebook is originated from the Kubeflow pipeline's [e2e-mnist](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/e2e-mnist) example that's running on Kubeflow 0.7. We have modified this notebook to run with Kubeflow 1.x's user namespace separation with Tekton support. 

This pipeline contains 5 steps, it finds the best hyperparameter using Katib, creates PVC for storing models, processes the hyperparameter results, distributedly trains the model on TFJob with the best hyperparameter using more iterations, and finally serves the model using KFServing. You can visit this [medium blog](https://medium.com/@liuhgxa/an-end-to-end-use-case-by-kubeflow-b2f72b0b587) for more details on this pipeline.

To run this pipeline, make sure your cluster has at least 16 cpu and 32GB in total. Otherwise some jobs might not able to run because TFJob needs to run 4 TensorFlow pods in parallel for distributed training.

## Prerequisites 
- Install [Kubeflow 1.0.2+](https://www.kubeflow.org/docs/started/getting-started/) and connect the cluster to the current shell with `kubectl`
- Install [kfp-tekton](/sdk/README.md#steps) SDK
- Install the necessary Python packages for running Jupyter notebook.
    ```shell
    pip install jupyter numpy Pillow
    ```
- Make sure the default storageclass supports ReadWriteMany in order to run distributed training. e.g. For IBM Cloud, you can run the following commands.
    ```shell
    new_storageclass=ibmc-file-gold
    old_storageclass=ibmc-block-gold
    kubectl patch storageclass ${new_storageclass} -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
    kubectl patch storageclass ${old_storageclass} -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"false"}}}'
    
    # Check is the default storageclass became the new_storageclass
    kubectl get storageclass | grep "(default)"
    ```

## Instructions

Once you have completed all the prerequisites for this example, then you can start the Jupyter server in this directory and click on the `mnist.ipynb` notebook. The notebook has step by step instructions for running the KFP Tekton pipeline.
```
jupyter notebook
```

## Acknowledgements

Thanks [Hougang Liu](https://github.com/hougangliu) for creating the original e2e-mnist notebook.
