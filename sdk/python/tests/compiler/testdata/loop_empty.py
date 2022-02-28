from kfp import dsl
from kfp_tekton.compiler import TektonCompiler
from kfp_tekton.tekton import Loop


class Coder:
    def empty(self):
        return ""


TektonCompiler._get_unique_id_code = Coder.empty


@dsl.pipeline("empty-loop")
def loop_empty(start: int = 0, end: int = 5):
  with Loop.range(start, end):
    pass


if __name__ == '__main__':
  TektonCompiler().compile(loop_empty, __file__.replace('.py', '.yaml'))

