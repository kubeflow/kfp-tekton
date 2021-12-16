# Copyright 2021 kubeflow.org
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from typing import List, Iterable, Union, Optional, TypeVar
from kfp.dsl import _pipeline_param, _for_loop
from kfp import dsl
from kfp import components
from kfp.dsl._pipeline_param import ConditionOperator
from kfp_tekton.compiler._k8s_helper import sanitize_k8s_name
from kfp_tekton.compiler._op_to_template import TEKTON_BASH_STEP_IMAGE


CEL_EVAL_IMAGE = "aipipeline/cel-eval:latest"
ANY_SEQUENCER_IMAGE = "dspipelines/any-sequencer:latest"
DEFAULT_CONDITION_OUTPUT_KEYWORD = "outcome"
TEKTON_CUSTOM_TASK_IMAGES = [CEL_EVAL_IMAGE]
LOOP_PIPELINE_NAME_LENGTH = 40
LOOP_GROUP_NAME_LENGTH = 16
_Num = TypeVar('_Num', int, float)


def AnySequencer(any: Iterable[Union[dsl.ContainerOp, ConditionOperator]],
                name: str = None, statusPath: str = None,
                skippingPolicy: str = None, errorPolicy: str = None,
                image: str = ANY_SEQUENCER_IMAGE):
    """A containerOp that will proceed when any of the dependent containerOps completed
       successfully

    Args:
        name: The name of the containerOp. It does not have to be unique within a pipeline
                because the pipeline will generate a unique new name in case of conflicts.

        any: List of `Conditional` containerOps that deploy together with the `main`
                containerOp, or the condtion that must meet to continue.

        statusPath: The location to write the output stauts

        skippingPolicy: Determines for the Any Sequencer reacts to
                no-dependency-condition-matching case. Values can be one of `skipOnNoMatch`
                or `errorOnNoMatch`, a status with value "Skipped" will be generated and the
                exit status will still be succeeded on `skipOnNoMatch`.

        errorPolicy: The standard field, either `failOnError` or `continueOnError`. On
                `continueOnError`, a status with value "Failed" will be generated
                but the exit status will still be succeeded. For `Fail_on_error` the
                Any Sequencer should truly fail in the Tekton terms, as it does now.

        image: The image to implement the any sequencer logic. Default to dspipelines/any-sequencer:latest.
    """
    arguments = [
                "--namespace",
                "$(context.pipelineRun.namespace)",
                "--prName",
                "$(context.pipelineRun.name)"
            ]
    tasks_list = []
    condition_list = []
    file_outputs = None
    for cop in any:
        if isinstance(cop, dsl.ContainerOp):
            cop_name = sanitize_k8s_name(cop.name)
            tasks_list.append(cop_name)
        elif isinstance(cop, ConditionOperator):
            condition_list.append(cop)
    if len(tasks_list) > 0:
        task_list_str = "\'" + ",".join(tasks_list) + "\'"
        arguments.extend(["--taskList", task_list_str])
    if statusPath is not None:
        file_outputs = '{outputPath: %s}' % statusPath
        arguments.extend(["--statusPath", file_outputs])
        if skippingPolicy is not None:
            assert skippingPolicy == "skipOnNoMatch" or skippingPolicy == "errorOnNoMatch"
            arguments.extend(["--skippingPolicy", skippingPolicy])
        if errorPolicy is not None:
            assert errorPolicy == "continueOnError" or errorPolicy == "failOnError"
            arguments.extend(["--errorPolicy", errorPolicy])
    conditonArgs = processConditionArgs(condition_list)
    arguments.extend(conditonArgs)

    AnyOp_yaml = '''\
    name: %s
    description: 'Proceed when any of the dependents completed successfully'
    outputs:
    - {name: %s, description: 'The output file to create the status'}
    implementation:
        container:
            image: %s
            command: [any-task]
            args: [%s]
    ''' % (name, statusPath, image, ",".join(arguments))
    AnyOp_template = components.load_component_from_text(AnyOp_yaml)
    AnyOp = AnyOp_template()
    return AnyOp


def processOperand(operand) -> (str, str):
    if isinstance(operand, dsl.PipelineParam):
        return "results_" + sanitize_k8s_name(operand.op_name) + "_" + sanitize_k8s_name(operand.name), operand.op_name
    else:
        # Do the same as in _get_super_condition_template to check whehter it's int
        try:
            operand = int(operand)
        except:
            operand = '\'' + str(operand) + '\''
        return operand, None


def processCondition(condition: ConditionOperator) -> str:
    op1, taskName1 = processOperand(condition.operand1)
    op2, taskName2 = processOperand(condition.operand2)
    if taskName1 is not None and taskName2 is not None:
        assert taskName1 == taskName2, "The result for condition must come from the same task."
    assert taskName1 is not None or taskName2 is not None, "Must at least contain one result in one condition for a task."
    conditionStr = f"{op1} {condition.operator} {op2}"
    return conditionStr


def processConditionArgs(conditions: List[ConditionOperator]) -> List[str]:
    conditionArgs = []
    for condition in conditions:
        conditionStr = processCondition(condition)
        conditionArgs.extend(["-c", conditionStr])
    return conditionArgs


def after_any(any: Iterable[Union[dsl.ContainerOp, ConditionOperator]], name: str = None,
              statusPath: str = None, skippingPolicy: str = None, errorPolicy: str = None):
    '''
    The function adds a new AnySequencer and connects the given op to it
    '''
    seq = AnySequencer(any, name, statusPath, skippingPolicy, errorPolicy)

    def _after_components(cop):
        cop.after(seq)
        return cop
    return _after_components


def CEL_ConditionOp(condition_statement):
    '''A containerOp template for CEL and converts it into Tekton custom task
    during Tekton compiliation.

    Args:
        condition_statement: CEL expression statement using string and/or pipeline params.
    '''
    ConditionOp_yaml = '''\
    name: 'condition-cel'
    description: 'Condition Operation using Common Expression Language'
    inputs:
    - {name: condition_statement, type: String, description: 'Condition statement', default: ''}
    outputs:
    - {name: %s, type: String, description: 'Default condition output'}
    implementation:
        container:
            image: %s
            command: ['sh', '-c']
            args: [
            '--apiVersion', 'cel.tekton.dev/v1alpha1',
            '--kind', 'CEL',
            '--name', 'cel_condition',
            '--%s', {inputValue: condition_statement},
            {outputPath: %s},
            ]
    ''' % (DEFAULT_CONDITION_OUTPUT_KEYWORD,
           CEL_EVAL_IMAGE,
           DEFAULT_CONDITION_OUTPUT_KEYWORD,
           DEFAULT_CONDITION_OUTPUT_KEYWORD)
    ConditionOp_template = components.load_component_from_text(ConditionOp_yaml)
    ConditionOp = ConditionOp_template(condition_statement)
    ConditionOp.add_pod_annotation("valid_container", "false")
    return ConditionOp


def Break():
    '''A BreakOp template for Break Operation using PipelineLoop
    '''
    BreakOp_yaml = '''\
    name: 'pipelineloop-break-operation'
    description: 'Break Operation using PipelineLoop'
    implementation:
        container:
            image: %s
            command:
            - sh
            - -c
            - |
              echo "$0"
            args:
            - "break loop"
    ''' % (TEKTON_BASH_STEP_IMAGE)
    BreakOp_template = components.load_component_from_text(BreakOp_yaml)
    BreakOp = BreakOp_template()
    return BreakOp


class Loop(dsl.ParallelFor):

  @classmethod
  def sequential(self,
                 loop_args: _for_loop.ItemList):
    return Loop(loop_args=loop_args, parallelism=1)

  @classmethod
  def from_string(self,
                  loop_args: Union[str, _pipeline_param.PipelineParam],
                  separator: Optional[str] = None,
                  parallelism: Optional[int] = None):
    return Loop(loop_args=loop_args, separator=separator, parallelism=parallelism)

  @classmethod
  def range(self,
            a: _Num,
            b: _Num,
            c: Optional[_Num] = None,
            parallelism: Optional[int] = None):
    return Loop(start=a, step=b, end=c, parallelism=parallelism)

  def __init__(self,
               loop_args: Union[_for_loop.ItemList,
                                _pipeline_param.PipelineParam] = None,
               start: _Num = None,
               end: _Num = None,
               step: _Num = None,
               separator: Optional[str] = None,
               parallelism: Optional[int] = None):
    tekton_params = (start, end, step, separator)
    if loop_args and not [x for x in tekton_params if x is not None]:
        super().__init__(loop_args=loop_args, parallelism=parallelism)
    elif loop_args and separator:
        # TODO: implement loop separator DSL extension
        pass
    elif start and end:
        # TODO: implement loop start, end, step DSL extension
        pass
    else:
        raise("loop_args or start/end parameters are missing for 'Loop' class")
