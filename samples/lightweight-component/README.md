# Lightweight python components example
This notebook demonstrates how to compile and execute the pipeline using simple python functions, also known as creating pipeline through lightweight. This notebook is originated from the kubeflow pipeline's [lighweight-component](https://github.com/kubeflow/pipelines/tree/master/samples/core/lightweight_component) example that is running with the KFP API. We have modified this notebook to run with the Tekton compiler and submit the pipeline directly to the Tekton engine.

This pipeline contains these steps, it creates `add` and `my_divmod` functions, converts to Kubeflow pipeline operation, defines the pipeline with hard coded parameters, and runs the pipeline on Tekton. You can see the detail about the pipeline at [Build Lightweight Python Components](https://www.kubeflow.org/docs/pipelines/sdk/lightweight-python-components/)

## Prerequisites
- Install [KFP Tekton prerequisites](/samples/README.md)
- Install [Jupyter notebook](https://jupyter.org/install)

## Instructions

Once you have completed all the prerequisites for this example, then you can start the Jupyter server in this directory and click on the `lightweight_component.ipynb` notebook. The notebook has step by step instructions for running the KFP Tekton pipeline.
```
python -m jupyter notebook
```

Or, you can compile the pipeline directly with:
```
python calc_pipeline.py
```

## Acknowledgements

Thanks [Jiaxiao Zheng](https://github.com/numerology) for creating the original Lightweight_component notebook.
