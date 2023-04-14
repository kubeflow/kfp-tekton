# The data passing pipeline

The data passing pipeline is used to demonstrate the use of different type of data passing in KFP and make sure Tekton passes the KFP data passing conformance test.

## prerequisites
- Install [KFP Tekton prerequisites](/samples/README.md)

## Instructions
* Compile the flip-coin pipeline using the compiler inside the python code. The kfp-tekton SDK will produce a Tekton pipeline yaml definition in the same directory called `data_passing.yaml`.
    ```
    # Compile the python code
    python data_passing.py
    ```

Then, upload the `data_passing.yaml` file to the Kubeflow pipeline dashboard with Tekton Backend to run this pipeline.
