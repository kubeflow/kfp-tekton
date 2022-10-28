import kfp
import kfp.components as comp

#Define a Python function
def add(a: float, b: float) -> float:
   """Calculates sum of two arguments"""
   return a + b

add_op = comp.func_to_container_op(add)

#Advanced function
#Demonstrates imports, helper functions and multiple outputs
from typing import NamedTuple
def my_divmod(dividend: float, divisor:float) -> NamedTuple('MyDivmodOutput', [('quotient', float), ('remainder', float), ('mlpipeline_ui_metadata', 'UI_metadata'), ('mlpipeline_metrics', 'Metrics')]):
    """Divides two numbers and calculate  the quotient and remainder"""
    #Pip installs inside a component function.
    #NOTE: installs should be placed right at the beginning to avoid upgrading a package
    # after it has already been imported and cached by python
    import sys, subprocess;
    subprocess.run([sys.executable, '-m', 'pip', 'install', 'tensorflow==1.8.0'])

    #Imports inside a component function:
    import numpy as np

    #This function demonstrates how to use nested functions inside a component function:
    def divmod_helper(dividend, divisor):
        return np.divmod(dividend, divisor)

    (quotient, remainder) = divmod_helper(dividend, divisor)

    from tensorflow.python.lib.io import file_io
    import json

    # Exports a sample tensorboard:
    metadata = {
      'outputs' : [{
        'type': 'tensorboard',
        'source': 'gs://ml-pipeline-dataset/tensorboard-train',
      }]
    }

    # Exports two sample metrics:
    metrics = {
      'metrics': [{
          'name': 'quotient',
          'numberValue':  float(quotient),
        },{
          'name': 'remainder',
          'numberValue':  float(remainder),
        }]}

    from collections import namedtuple
    divmod_output = namedtuple('MyDivmodOutput', ['quotient', 'remainder', 'mlpipeline_ui_metadata', 'mlpipeline_metrics'])
    return divmod_output(quotient, remainder, json.dumps(metadata), json.dumps(metrics))

divmod_op = comp.func_to_container_op(my_divmod, base_image='tensorflow/tensorflow:1.11.0-py3')

import kfp.dsl as dsl
@dsl.pipeline(
   name='calculation-pipeline',
   description='A toy pipeline that performs arithmetic calculations.'
)
# Currently kfp-tekton doesn't support pass parameter to the pipelinerun yet, so we hard code the number here
def calc_pipeline(
   a:float = 7.0,
   b:float = 8.0,
   c:float = 17.0,
):
    #Passing pipeline parameter and a constant value as operation arguments
    add_task = add_op(a, 4) #Returns a dsl.ContainerOp class instance.

    #Passing a task output reference as operation arguments
    #For an operation with a single return value, the output reference can be accessed using `task.output` or `task.outputs['output_name']` syntax
    divmod_task = divmod_op(add_task.output, b)
    divmod_task.add_pod_annotation("tekton.dev/track_step_artifact", "true")

    #For an operation with a multiple return values, the output references can be accessed using `task.outputs['output_name']` syntax
    result_task = add_op(divmod_task.outputs['quotient'], c)

if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(calc_pipeline, __file__.replace('.py', '.yaml'))
