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

import kfp.compiler.main as kfp_compiler_main
import argparse
import sys
import os
from . import TektonCompiler

from .. import __version__


def parse_arguments():
  """Parse command line arguments."""

  parser = argparse.ArgumentParser()
  parser.add_argument('--py',
                      type=str,
                      help='local absolute path to a py file.')
  parser.add_argument('--package',
                      type=str,
                      help='local path to a pip installable python package file.')
  parser.add_argument('--function',
                      type=str,
                      help='The name of the function to compile if there are multiple.')
  parser.add_argument('--namespace',
                      type=str,
                      help='The namespace for the pipeline function')
  parser.add_argument('--output',
                      type=str,
                      required=True,
                      help='local path to the output workflow yaml file.')
  parser.add_argument('--disable-type-check',
                      action='store_true',
                      help='disable the type check, default is enabled.')

  args = parser.parse_args()
  return args


def _compile_pipeline_function(pipeline_funcs, function_name, output_path, type_check):
  if len(pipeline_funcs) == 0:
    raise ValueError('A function with @dsl.pipeline decorator is required in the py file.')

  if len(pipeline_funcs) > 1 and not function_name:
    func_names = [x.__name__ for x in pipeline_funcs]
    raise ValueError('There are multiple pipelines: %s. Please specify --function.' % func_names)

  if function_name:
    pipeline_func = next((x for x in pipeline_funcs if x.__name__ == function_name), None)
    if not pipeline_func:
      raise ValueError('The function "%s" does not exist. '
                       'Did you forget @dsl.pipeline decoration?' % function_name)
  else:
    pipeline_func = pipeline_funcs[0]

  TektonCompiler().compile(pipeline_func, output_path, type_check)


def compile_pyfile(pyfile, function_name, output_path, type_check):
  sys.path.insert(0, os.path.dirname(pyfile))
  try:
    filename = os.path.basename(pyfile)
    with kfp_compiler_main.PipelineCollectorContext() as pipeline_funcs:
      __import__(os.path.splitext(filename)[0])
    _compile_pipeline_function(pipeline_funcs, function_name, output_path, type_check)
  finally:
    del sys.path[0]


def main():
    print(f"KFP-Tekton Compiler {__version__}")
    args = parse_arguments()
    if ((args.py is None and args.package is None) or
        (args.py is not None and args.package is not None)):
        raise ValueError('Either --py or --package is needed but not both.')
    if args.py:
        compile_pyfile(args.py, args.function, args.output, not args.disable_type_check)
    else:
        if args.namespace is None:
            raise ValueError('--namespace is required for compiling packages.')
        # TODO: deprecated, but without the monkey-patch this will produce Argo YAML
        kfp_compiler_main.compile_package(args.package, args.namespace, args.function, args.output, not args.disable_type_check)
