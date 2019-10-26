import kfp.dsl as dsl
import kfp
from kfp import components
import kfp.compiler as compiler
import json

kfserving_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/master/components/kubeflow/kfserving/component.yaml')

@dsl.pipeline(
  name='kfserving pipeline',
  description='A pipeline for kfserving.'
)
def kfservingPipeline(
    action = 'update',
    model_name='tfx-taxi',
    default_model_uri='',
    canary_model_uri='',
    canary_model_traffic_percentage='0',
    namespace='default',
    framework='custom',
    default_custom_model_spec='{"name": "tfx-taxi", "image": "tomcli/tf-serving:latest", "port": "8501", "env": [{"name":"MODEL_NAME", "value": "tfx-taxi"},{"name":"MODEL_BASE_PATH", "value": "/models"}]}',
    canary_custom_model_spec='{}',
    autoscaling_target='0',
    kfserving_endpoint=''
):

    # define workflow
    kfserving = kfserving_op(action = action,
                             model_name=model_name,
                             default_model_uri=default_model_uri,
                             canary_model_uri=canary_model_uri,
                             canary_model_traffic_percentage=canary_model_traffic_percentage,
                             namespace=namespace,
                             framework=framework,
                             default_custom_model_spec=default_custom_model_spec,
                             canary_custom_model_spec=canary_custom_model_spec,
                             autoscaling_target=autoscaling_target,
                             kfserving_endpoint=kfserving_endpoint).set_image_pull_policy('Always')


if __name__ == '__main__':
    # Compile pipeline
    compiler.Compiler().compile(kfservingPipeline, 'custom.tar.gz')
