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
from typing import List, Iterable, Union, Optional, TypeVar, Text

from kfp.dsl import _pipeline_param, _for_loop, _pipeline
from kfp import dsl
from kfp import components
from kfp.dsl._for_loop import LoopArguments, ItemList
from kfp.dsl._pipeline_param import ConditionOperator, PipelineParam
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


class TektonLoopArguments(LoopArguments):
  def __init__(self,
               items: Union[str, ItemList, dsl.PipelineParam],
               code: Text,
               name_override: Optional[Text] = None,
               op_name: Optional[Text] = None,
               *args,
               **kwargs):
    if isinstance(items, str):
        # temporary list wrapping for validation to pass
        super().__init__([items], code, name_override, op_name, *args, **kwargs)
        self.items_or_pipeline_param = items
    else:
        super().__init__(items, code, name_override, op_name, *args, **kwargs)

  def to_str_for_task_yaml(self):
    if isinstance(self.items_or_pipeline_param, str):
      return self.items_or_pipeline_param
    else:
      raise ValueError(
          'You should only call this method on loop args which are string literals, '
          'not lists or pipeline param items.')


class Loop(dsl.ParallelFor):
  @classmethod
  def sequential(cls,
                 loop_args: _for_loop.ItemList):
    return cls(loop_args=loop_args, parallelism=1)

  @classmethod
  def from_string(cls,
                  loop_args: Union[str, _pipeline_param.PipelineParam],
                  separator: Optional[str] = None,
                  parallelism: Optional[int] = None):
    return cls(loop_args=loop_args, separator=separator, parallelism=parallelism)

  @classmethod
  def range(cls,
            start: Union[_Num, PipelineParam],
            end: Union[_Num, PipelineParam],
            step: Optional[Union[_Num, PipelineParam]] = None,
            parallelism: Optional[int] = None):
    return cls(start=start, step=step, end=end, parallelism=parallelism)

  def __init__(self,
               loop_args: Union[str,
                                _for_loop.ItemList,
                                _pipeline_param.PipelineParam] = None,
               start: Union[_Num, PipelineParam, None] = None,
               end: Union[_Num, PipelineParam, None] = None,
               step: Union[_Num, PipelineParam, None] = None,
               separator: Optional[str] = None,
               parallelism: Optional[int] = None):
    self.start = None
    self.end = None
    self.step = None
    if start and end:
        super().__init__(loop_args=["iteration"], parallelism=parallelism)
        self.start = start
        self.end = end
        self.step = step
    else:
        if loop_args is None and (start is None or end is None):
            raise RuntimeError("loop_args or start/end parameters are missing for 'Loop' class")

        def next_id():
            return str(_pipeline.Pipeline.get_default_pipeline().get_next_group_id())

        if isinstance(loop_args, str):
            # temporary list wrapping for validation to pass
            super().__init__(loop_args=[loop_args], parallelism=parallelism)
            self.loop_args = TektonLoopArguments(
                loop_args,
                code=next_id(),
            )
            self.items_is_string = True
        else:
            super().__init__(loop_args=loop_args, parallelism=parallelism)
            self.items_is_string = False

        self.separator = None
        if separator is not None:
            self.separator = PipelineParam(
                name=LoopArguments._make_name(next_id()),
                value=separator
            )
