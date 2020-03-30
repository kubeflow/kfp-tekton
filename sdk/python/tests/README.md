## Tests for the Compiler


The test listed here can be used to compile all Python DSL pipelines in the KFP compiler testdata folder and generate a report card. As you are doing a PR to address functionality gaps in compiler, please run this test to ensure that they have been addressed. Please note that even if a Kubeflow Pipeline Python DSL script may pass the compilation with the KFP-Tekton compiler successfully, the produced Tekton YAML might not be valid or contain all of the intended functionality as the equivalent Argo YAML produced by the KFP compiler. For that, best way is to take the compiled YAML and run it on Tekton directly.

### Running the test

    - `./sdk/python/tests/test_kfp_samples.sh`

You should see an output similar to the one below, outlining which samples have passed and which are failing.               

```bash
SUCCESS: add_pod_env.py
SUCCESS: artifact_location.py
FAILURE: basic.py
FAILURE: basic_no_decorator.py
SUCCESS: coin.py
FAILURE: compose.py
SUCCESS: default_value.py
FAILURE: input_artifact_raw_value.py
SUCCESS: loop_over_lightweight_output.py
SUCCESS: param_op_transform.py
FAILURE: param_substitutions.py
SUCCESS: pipelineparams.py
SUCCESS: recursive_do_while.py
SUCCESS: recursive_while.py
FAILURE: resourceop_basic.py
SUCCESS: sidecar.py
SUCCESS: timeout.py
SUCCESS: volume.py
FAILURE: volume_snapshotop_rokurl.py
FAILURE: volume_snapshotop_sequential.py
FAILURE: volumeop_basic.py
FAILURE: volumeop_dag.py
FAILURE: volumeop_parallel.py
FAILURE: volumeop_sequential.py
SUCCESS: withitem_basic.py
SUCCESS: withitem_nested.py
SUCCESS: withparam_global.py
SUCCESS: withparam_global_dict.py
SUCCESS: withparam_output.py
SUCCESS: withparam_output_dict.py

Success: 18
Failure: 12
Total:   30

The compilation status report was stored in /kfp-tekton/sdk/python/tests/test_kfp_samples_report.txt
The accumulated console logs can be found in /kfp-tekton/temp/test_kfp_samples_output.txt
```

Goal should be to have all the 30 tests pass before we have a fair degree of confidence that the compile can handle a fair number of pipelines.

