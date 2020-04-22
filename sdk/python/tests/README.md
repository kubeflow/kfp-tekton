## Tests for the Compiler


The test listed here can be used to compile all Python DSL pipelines in the KFP compiler `testdata` folder and 
generate a report card. As you are working a PR to address functionality gaps in the compiler, please run this test to
update the `FAILURE`s which have been addressed.

Please note that even if a Kubeflow Pipeline Python DSL script may pass the compilation with the KFP-Tekton compiler
successfully, the produced Tekton YAML might not be valid or not contain all of the intended functionality as the 
equivalent Argo YAML produced by the KFP compiler. For that, the best way is to take the compiled YAML and run it
on Tekton directly.

### Running the tests

`./sdk/python/tests/test_kfp_samples.sh`

You should see an output similar to the one below, outlining which test scripts have passed and which are failing:

```YAML
SUCCESS: add_pod_env.py
SUCCESS: artifact_location.py
SUCCESS: basic.py
/Users/animeshsingh/go/src/github.com/kubeflow/kfp-tekton/.venv/bin/python3: No module named test_util
FAILURE: coin.py
/Users/animeshsingh/go/src/github.com/kubeflow/kfp-tekton/.venv/bin/python3: No module named test_util
SUCCESS: default_value.py
FAILURE: input_artifact_raw_value.py
FAILURE: loop_over_lightweight_output.py
SUCCESS: param_op_transform.py
SUCCESS: param_substitutions.py
SUCCESS: pipelineparams.py
FAILURE: recursive_do_while.py
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

Success: 20
Failure: 8
Total:   28

Compilation status report:   sdk/python/tests/test_kfp_samples_report.txt
Accumulated compiler logs:   temp/test_kfp_samples_output.txt
Compiled Tekton YAML files:  temp/tekton_compiler_output/
```

The goal should be to have all the 30 tests pass before we can have a degree of confidence that the compiler can handle
a fair number of pipelines.

