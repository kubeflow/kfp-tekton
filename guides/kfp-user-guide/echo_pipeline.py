from kfp import dsl
def echo_op():
    return dsl.ContainerOp(
        name='echo',
        image='busybox',
        command=['sh', '-c'],
        arguments=['echo "Got scheduled"']
    )

@dsl.pipeline(
    name='echo',
    description='echo pipeline'
)
def echo_pipeline(
):
    echo = echo_op()

if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(echo_pipeline, 'echo_pipeline.yaml')
