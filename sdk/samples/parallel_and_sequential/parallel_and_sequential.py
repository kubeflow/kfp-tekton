from kfp_tekton.compiler import TektonCompiler
from kfp import dsl


def gcs_download_op(url):
    return dsl.ContainerOp(
        name='GCS - Download',
        image='google/cloud-sdk:216.0.0',
        command=['sh', '-c'],
        arguments=['gsutil cat $0 | tee $1', url, '/tmp/results.txt'],
    )


def echo_op(text):
    return dsl.ContainerOp(
        name='echo',
        image='library/bash:4.4.23',
        command=['sh', '-c'],
        arguments=['echo "$0"', text]
    )


@dsl.pipeline(
    name='Parallel and Sequential pipeline',
    description='A pipeline with two parallel tasks followed by a sequential task.'
)
def parallel_and_sequential_pipeline(url1='gs://ml-pipeline-playground/shakespeare1.txt',
                                     url2='gs://ml-pipeline-playground/shakespeare2.txt',
                                     path='/tmp/results.txt'):
    """A pipeline with two parallel tasks followed by a sequential task."""

    download_task1 = gcs_download_op(url1)
    download_task2 = gcs_download_op(url2)
    echo_task = echo_op(path)

    echo_task.after(download_task1).after(download_task2)


if __name__ == '__main__':
    TektonCompiler().compile(parallel_and_sequential_pipeline, 'parallel_and_sequential.yaml')
