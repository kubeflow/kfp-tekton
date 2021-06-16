from kfp import dsl
from kfp.components import load_component_from_text
from kfp_tekton.tekton import CEL_ConditionOp


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


class CEL_Condition(dsl.Condition):
  def __init__(self, pred: str, name: str = None):
    super().__init__(CEL_ConditionOp(pred).output == 'true', name)


def CEL_ExprOp(expr: str):
  return CEL_ConditionOp(expr)


@dsl.pipeline("double-recursion test")
def double_recursion_test(until_a: int = 4, until_b: int = 3):
  @dsl.graph_component
  def recur_a(i: int, until_a: int):
    @dsl.graph_component
    def recur_b(j: int, until_b: int):
      print_op = PrintOp(f"Iter A: {i}, B: {j}")
      incr_j = CEL_ExprOp(f"{j} + 1").after(print_op).output
      with CEL_Condition(f"{incr_j} < {until_b}"):
        recur_b(incr_j, until_b)

    start_b = CEL_ExprOp("0").output
    with CEL_Condition(f"{start_b} < {until_b}"):
      b = recur_b(start_b, until_b)

    incr_i = CEL_ExprOp(f"{i} + 1").after(b).output
    with CEL_Condition(f"{incr_i} < {until_a}"):
      recur_a(incr_i, until_a)

  start_a = CEL_ExprOp("0").output
  with CEL_Condition(f"{start_a} < {until_a}"):
    recur_a(start_a, until_a)


if __name__ == '__main__':
  from kfp_tekton.compiler import TektonCompiler as Compiler
  Compiler().compile(double_recursion_test, __file__.replace('.py', '.yaml'))
