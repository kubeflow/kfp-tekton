# TODO: from KFP 1.3.0, need to implement for kfp_tekton.compiler

import kfp.dsl as dsl


@dsl.graph_component
def echo1_graph_component(text1):
  dsl.ContainerOp(
      name='echo1-task1',
      image='library/bash:4.4.23',
      command=['sh', '-c'],
      arguments=['echo "$0"', text1])


@dsl.graph_component
def echo2_graph_component(text2):
  dsl.ContainerOp(
      name='echo2-task1',
      image='library/bash:4.4.23',
      command=['sh', '-c'],
      arguments=['echo "$0"', text2])


@dsl.pipeline()
def opsgroups_pipeline(text1='message 1', text2='message 2'):
  step1_graph_component = echo1_graph_component(text1)
  step2_graph_component = echo2_graph_component(text2)
  step2_graph_component.after(step1_graph_component)
