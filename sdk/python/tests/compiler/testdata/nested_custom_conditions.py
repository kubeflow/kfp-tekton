from kfp import dsl
from kfp.components import load_component_from_text
from kfp_tekton.tekton import TEKTON_CUSTOM_TASK_IMAGES
from kfp import components


PrintOp = load_component_from_text("""
  name: print
  inputs:
  - name: msg
  outputs:
  - name: stdout
  implementation:
    container:
      image: alpine:3.6
      command:
      - concat:
        - "echo "
        - { inputValue: msg }
""")

CEL_EXPRS_IMAGE = "cel-exprs/image:latest"
TEKTON_CUSTOM_TASK_IMAGES = TEKTON_CUSTOM_TASK_IMAGES.append(CEL_EXPRS_IMAGE)


def CEL_Exprs(**conds):
  ConditionOp_yaml = '''\
  name: 'cel-exprs'
  inputs:
  - {name: ab, type: String, description: 'Condition statement ab', default: ''}
  - {name: bc, type: String, description: 'Condition statement bc', default: ''}
  outputs:
  - {name: ab, type: String, description: 'Default condition ab output'}
  - {name: bc, type: String, description: 'Default condition bc output'}
  implementation:
      container:
          image: %s
          command: ['sh', '-c']
          args: [
          '--apiVersion', 'custom.tekton.dev/v1alpha1',
          '--kind', 'CelExprs',
          '--name', 'cel_exprs',
          '--ab', {inputValue: ab},
          '--bc', {inputValue: bc},
          {outputPath: ab},
          {outputPath: bc}
          ]
  ''' % (CEL_EXPRS_IMAGE)
  ConditionOp_template = components.load_component_from_text(ConditionOp_yaml)
  conds_values = list(conds.values())
  op = ConditionOp_template(conds_values[0], conds_values[1])
  op.add_pod_annotation("valid_container", "false")
  return op


@dsl.pipeline("nested-condition-test")
def nested_condition_test(a: int, b: int, c: int):
  op = CEL_Exprs(
    ab=f"{a} < {b}",
    bc=f"{b} < {c}",
  )
  with dsl.Condition(op.outputs["ab"] == 'true'):
    with dsl.Condition(op.outputs["bc"] == 'true'):
      print_op = PrintOp(f"{a} < {b} < {c}")


if __name__ == '__main__':
  from kfp_tekton.compiler import TektonCompiler as Compiler
  Compiler().compile(nested_condition_test, __file__.replace('.py', '.yaml'))
