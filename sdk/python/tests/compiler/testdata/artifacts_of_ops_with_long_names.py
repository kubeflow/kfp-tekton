# Copyright 2021 kubeflow.org.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import os
from random import choice
from string import ascii_lowercase
from typing import Optional, List
import pytest

from kfp import dsl
from kfp.components import load_component_from_text


def random_chars_of_len(length: int) -> str:
  return ''.join(
    (choice(ascii_lowercase) for i in range(length))
  )


def pipeline_for_len(op_name_length: int, op_name_base: str = ''):
  len_to_add = 0
  if len(op_name_base) < op_name_length:
    len_to_add = op_name_length - len(op_name_base)
  op_name_base += random_chars_of_len(len_to_add)

  op_name = op_name_base[:op_name_length]

  component_spec = f"""
    name: {op_name}
    inputs:
    - name: i
      type: Integer
    outputs:
    - name: incr_i
      type: Integer
    - name: sq_i
      type: Integer
    implementation:
      container:
        image: sth
        command:
        - whatever
        - outputPath: sq_i
        - outputPath: incr_i
  """

  op = load_component_from_text(component_spec)

  @dsl.pipeline("main-pipeline")
  def main_pipeline(i: int):
    op(i)
  return main_pipeline


lengths = [20, 55, 80]


def main(fdir: Optional[str] = None) -> List[str]:
  fpaths: List[str] = list()

  from kfp_tekton.compiler import TektonCompiler as Compiler
  max_length = max(lengths)
  op_name_base = random_chars_of_len(max_length)

  for length in lengths:
    main_pipeline = pipeline_for_len(length, op_name_base)
    fpath = __file__.replace('.py', f'-{length}.yaml')
    fname = os.path.basename(fpath)
    if fdir is not None:
      fpath = os.path.join(fdir, fname)
    if length > 57:
      # Op name cannot be more than 57 characters
      with pytest.raises(ValueError):
        Compiler().compile(main_pipeline, fpath)
    else:
      Compiler().compile(main_pipeline, fpath)
      fpaths.append(fpath)

  return fpaths


if __name__ == '__main__':
  main()
