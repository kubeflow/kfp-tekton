# Copyright 2020-2021 kubeflow.org
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


import kfp
import os
from kfp import dsl
from kfp.components import create_component_from_func, OutputPath

cwd = os.path.dirname(__file__)
filesystem_component_root = 'components/filesystem'
get_subdir_op = kfp.components.load_component_from_file(os.path.join(
    cwd, '../../../../../', filesystem_component_root, 'get_subdirectory/component.yaml'))
list_item_op_1 = kfp.components.load_component_from_file(os.path.join(
    cwd, '../../../../../', filesystem_component_root, 'list_items/component.yaml'))
list_item_op_2 = kfp.components.load_component_from_file(
    os.path.join(cwd, 'create_component_from_func_component.yaml'))


@dsl.pipeline(name='Create component from function')
def create_component_pipeline():
    @create_component_from_func
    def produce_dir_with_files_python_op(output_dir_path: OutputPath(), num_files: int = 10):
        import os
        os.makedirs(os.path.join(output_dir_path, 'subdir'), exist_ok=True)
        for i in range(num_files):
            file_path = os.path.join(output_dir_path, 'subdir', str(i) + '.txt')
            with open(file_path, 'w') as f:
                f.write(str(i))

    produce_dir_python_task = produce_dir_with_files_python_op(num_files=15)

    get_subdir_task = get_subdir_op(
        # Input name "Input 1" is converted to pythonic parameter name "input_1"
        directory=produce_dir_python_task.output,
        subpath="subdir"
    )

    list_items_task_1 = list_item_op_1(
        # Input name "Input 1" is converted to pythonic parameter name "input_1"
        directory=get_subdir_task.output
    )

    list_items_task_2 = list_item_op_2(
        # Input name "Input 1" is converted to pythonic parameter name "input_1"
        directory1=get_subdir_task.output,
        directory2=produce_dir_python_task.output
    )


# General by kfp-tekton
if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(create_component_pipeline, __file__.replace('.py', '.yaml'))
