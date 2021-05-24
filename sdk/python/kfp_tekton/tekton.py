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

from typing import List, Iterable, Union
from kfp import dsl
from kfp.dsl._container_op import ContainerOp
from kfp.dsl._pipeline_param import ConditionOperator
from kfp_tekton.compiler._k8s_helper import sanitize_k8s_name


CEL_EVAL_IMAGE = "aipipeline/cel-eval:latest"
ANY_SEQUENCER_IMAGE = "dspipelines/any-sequencer:latest"
DEFAULT_CONDITION_OUTPUT_KEYWORD = "outcome"
TEKTON_CUSTOM_TASK_IMAGES = [CEL_EVAL_IMAGE]


class AnySequencer(ContainerOp):
    """A containerOp that will proceed when any of the dependent containerOps completed
       successfully

    Args:
        name: The name of the containerOp. It does not have to be unique within a pipeline
                because the pipeline will generate a unique new name in case of conflicts.

        any: List of `Conditional` containerOps that deploy together with the `main`
                containerOp, or the condtion that must meet to continue.
    """
    def __init__(self,
                 any: Iterable[Union[dsl.ContainerOp, ConditionOperator]],
                 name: str = None, statusPath: str = None,
                 skippingPolicy: str = None, errorPolicy: str = None):
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
            task_list_str = ",".join(tasks_list)
            arguments.extend(["--taskList", task_list_str])
        if statusPath is not None:
            file_outputs = {"status": statusPath}
            arguments.extend(["--statusPath", statusPath])
            if skippingPolicy is not None:
                assert skippingPolicy == "skipOnNoMatch" or skippingPolicy == "errorOnNoMatch"
                arguments.extend(["--skippingPolicy", skippingPolicy])
            if errorPolicy is not None:
                assert errorPolicy == "continueOnError" or errorPolicy == "failOnError"
                arguments.extend(["--errorPolicy", errorPolicy])

        conditonArgs = processConditionArgs(condition_list)
        arguments.extend(conditonArgs)

        super().__init__(
            name=name,
            image=ANY_SEQUENCER_IMAGE,
            file_outputs=file_outputs,
            command="any-taskrun",
            arguments=arguments,
        )


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
    ConditionOp = dsl.ContainerOp(
            name="condition-cel",
            image=CEL_EVAL_IMAGE,
            command=["sh", "-c"],
            arguments=["--apiVersion", "cel.tekton.dev/v1alpha1",
                       "--kind", "CEL",
                       "--name", "cel_condition",
                       "--%s" % DEFAULT_CONDITION_OUTPUT_KEYWORD, condition_statement],
            file_outputs={DEFAULT_CONDITION_OUTPUT_KEYWORD: '/tmp/tekton'}
        )
    ConditionOp.add_pod_annotation("valid_container", "false")
    return ConditionOp
