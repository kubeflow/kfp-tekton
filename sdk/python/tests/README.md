# Compiler Status Report

This report shows the compilation status for all Python DSL pipeline scripts in the KFP compiler
[`testdata`](https://github.com/kubeflow/pipelines/tree/master/sdk/python/tests/compiler/testdata)
folder.

As you are working on a PR to address functionality gaps in the compiler, please run this report to
update the compile `FAILURE`s which have been addressed by your code changes.

Please note that even if a Kubeflow Pipeline Python DSL script passes the compilation with the
KFP-Tekton compiler successfully, the produced Tekton YAML might not be valid or may not contain all
of the intended functionality as the equivalent Argo YAML produced by the KFP compiler.
To verify that the compiled YAML is valid and that the pipeline can be executed successfully it
needs to be deployed and run on a Tekton cluster.

## Generating the Compiler Status Report

To update this document, regenerate the report by running this script:

    ./test_kfp_samples.sh

or run this command from the project root directory:

    make report

You should see an output similar to the one below, outlining which test scripts have passed and
which are failing:

```YAML
KFP clone version: 1.8.4
KFP Python SDK version(s):
kfp                      1.8.14
kfp-pipeline-spec        0.1.16
kfp-server-api           1.8.5
kfp-tekton               1.3.1

SUCCESS: add_pod_env.py
SUCCESS: artifact_passing_using_volume.py
SUCCESS: basic.py
SUCCESS: basic_no_decorator.py
SUCCESS: coin.py
SUCCESS: compose.py
SUCCESS: default_value.py
SUCCESS: input_artifact_raw_value.py
SUCCESS: loop_over_lightweight_output.py
SUCCESS: opsgroups.py
SUCCESS: parallelfor_item_argument_resolving.py
SUCCESS: parallelfor_pipeline_param_in_items_resolving.py
SUCCESS: param_op_transform.py
SUCCESS: param_substitutions.py
SUCCESS: pipelineparams.py
SUCCESS: recursive_do_while.py
SUCCESS: recursive_while.py
SUCCESS: resourceop_basic.py
SUCCESS: sidecar.py
SUCCESS: timeout.py
SUCCESS: two_step.py
FAILURE: uri_artifacts.py
SUCCESS: volume.py
SUCCESS: volume_snapshotop_rokurl.py
SUCCESS: volume_snapshotop_sequential.py
SUCCESS: volumeop_basic.py
SUCCESS: volumeop_dag.py
SUCCESS: volumeop_parallel.py
SUCCESS: volumeop_sequential.py
SUCCESS: withitem_basic.py
SUCCESS: withitem_nested.py
SUCCESS: withparam_global.py
SUCCESS: withparam_global_dict.py
SUCCESS: withparam_output.py
SUCCESS: withparam_output_dict.py

Compilation status for testdata DSL scripts:

  Success: 34
  Failure: 1
  Total:   35

Overall success rate: 34/35 = 97%

Compilation status report:   sdk/python/tests/test_kfp_samples_report.txt
Accumulated compiler logs:   temp/test_kfp_samples_output.txt
Compiled Tekton YAML files:  temp/tekton_compiler_output/
```

The goal is to have all the `31` tests pass before we can have a degree of confidence that the
compiler can handle a fair number of pipelines.


## Summary Report for all KFP Sample DSL Scripts

For a more comprehensive report about the compilation status for all of the Python DSL scripts
found in the [`kubeflow/pipelines`](https://github.com/kubeflow/pipelines/) repository you may
run this report:

    ./test_kfp_samples.sh \
        --include-all-samples \
        --dont-list-files

This will include all `core/samples`, 3rd-party contributed samples, tutorials, as well as
the compiler `testdata`.

```YAML
Compilation status for testdata DSL scripts:

  Success: 25
  Failure: 6
  Total:   31

Compilation status for core samples:

  Success: 18
  Failure: 4
  Total:   22

Compilation status for 3rd-party contributed samples:

  Success: 25
  Failure: 7
  Total:   32

Overall success rate: 71/88 = 81%
```

When the `--print-error-details` flag is used, a summary of all the compilation errors is appended
to the console output -- sorted by their respective number of occurrences:

    ./test_kfp_samples.sh -a -s --print-error-details

```YAML
...

Overall success rate: 71/88 = 81%

Occurences of NotImplementedError:
   8 dynamic params are not yet implemented

Occurences of other Errors:
   2 ValueError: These Argo variables are not supported in Tekton Pipeline: {{workflow.uid}}
   1 ValueError: These Argo variables are not supported in Tekton Pipeline: {{workflow.name}}, {{pod.name}}
   1 ValueError: These Argo variables are not supported in Tekton Pipeline: {{workflow.name}}
   1 ValueError: These Argo variables are not supported in Tekton Pipeline: {{pod.name}}, {{workflow.uid}}
   1 ValueError: These Argo variables are not supported in Tekton Pipeline: {{pod.name}}, {{workflow.name}}
   1 ValueError: There are multiple pipelines: ['flipcoin_pipeline', 'flipcoin_exit_pipeline']. Please specify --function.
```

## Disclaimer

**Note:** The reports above were created for the pipeline scripts found in KFP SDK version `1.8.14` since
the `kfp_tekton` `1.3.1` compiler code is based on the `kfp` SDK compiler version greater than or equals to
`1.8.10` and less than `1.8.15`.
