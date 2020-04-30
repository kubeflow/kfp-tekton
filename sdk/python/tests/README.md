# Compiler Status Report

This report shows the compilation status for all Python DSL pipeline scripts in the KFP compiler
[`testdata`](https://github.com/kubeflow/pipelines/tree/master/sdk/python/tests/compiler/testdata) folder. 

As you are working on a PR to address functionality gaps in the compiler, please run this report to update the
compile `FAILURE`s which have been addressed by your code changes.

Please note that even if a Kubeflow Pipeline Python DSL script passes the compilation with the KFP-Tekton compiler
successfully, the produced Tekton YAML might not be valid or may not contain all of the intended functionality as the 
equivalent Argo YAML produced by the KFP compiler. To verify that the compiled YAML is valid and that the pipeline can
be executed successfully it needs to be deployed and run on a Tekton cluster.

## Generating the Compiler Status Report

To update this document, regenerate the report by running this script:

    ./test_kfp_samples.sh

or run this command from the project root directory:

    make report

You should see an output similar to the one below, outlining which test scripts have passed and which are failing:

```YAML
SUCCESS: add_pod_env.py
SUCCESS: artifact_location.py
SUCCESS: basic.py
SUCCESS: basic_no_decorator.py
SUCCESS: coin.py
SUCCESS: compose.py
SUCCESS: default_value.py
SUCCESS: input_artifact_raw_value.py
FAILURE: loop_over_lightweight_output.py
SUCCESS: param_op_transform.py
SUCCESS: param_substitutions.py
SUCCESS: pipelineparams.py
SUCCESS: recursive_do_while.py
SUCCESS: recursive_while.py
SUCCESS: resourceop_basic.py
SUCCESS: sidecar.py
SUCCESS: timeout.py
SUCCESS: volume.py
SUCCESS: volume_snapshotop_rokurl.py
SUCCESS: volume_snapshotop_sequential.py
SUCCESS: volumeop_basic.py
SUCCESS: volumeop_dag.py
SUCCESS: volumeop_parallel.py
SUCCESS: volumeop_sequential.py
SUCCESS: withitem_basic.py
SUCCESS: withitem_nested.py
FAILURE: withparam_global.py
FAILURE: withparam_global_dict.py
FAILURE: withparam_output.py
FAILURE: withparam_output_dict.py

Success: 25
Failure: 5
Total:   30

Compilation status report:   sdk/python/tests/test_kfp_samples_report.txt
Accumulated compiler logs:   temp/test_kfp_samples_output.txt
Compiled Tekton YAML files:  temp/tekton_compiler_output/
```

The goal is to have all the 30 tests pass before we can have a degree of confidence that the compiler can handle
a fair number of pipelines.
