# Copyright 2020 kubeflow.org
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

import sys
import traceback

from .compiler import TektonCompiler


def monkey_patch():
    """
    Overriding (replacing) selected methods/function in the KFP SDK compiler package.
    This is a temporary hack during early development of the KFP-Tekton compiler.
    """
    import kfp
    from kfp.compiler._data_passing_rewriter import fix_big_data_passing
    from kfp.compiler._k8s_helper import convert_k8s_obj_to_json
    from kfp.compiler._op_to_template import _op_to_template, _process_base_ops
    from kfp.compiler.compiler import Compiler as KFPCompiler

    from ._data_passing_rewriter import fix_big_data_passing as tekton_fix_big_data_passing
    from ._k8s_helper import convert_k8s_obj_to_json as tekton_convert_k8s_obj_to_json
    from ._op_to_template import _op_to_template as tekton_op_to_template
    from ._op_to_template import _process_base_ops as tekton_process_base_ops
    from .compiler import TektonCompiler

    kfp.compiler._data_passing_rewriter.fix_big_data_passing = tekton_fix_big_data_passing
    kfp.compiler._k8s_helper.convert_k8s_obj_to_json = tekton_convert_k8s_obj_to_json
    kfp.compiler._op_to_template._op_to_template = tekton_op_to_template
    kfp.compiler._op_to_template._process_base_ops = tekton_process_base_ops
    KFPCompiler._create_dag_templates = TektonCompiler._create_dag_templates
    KFPCompiler._create_pipeline_workflow = TektonCompiler._create_pipeline_workflow


try:
    # print("Applying monkey patch")
    monkey_patch()
except Exception as error:
    traceback.print_exc()
    print("Failed to apply monkey patch")
    sys.exit(1)
