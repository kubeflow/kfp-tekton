import kfp.dsl as dsl
import kfp.components as comp
from kfp_tekton.compiler import TektonCompiler
from kfp_tekton.k8s_client_helper import env_from_secret


def deploy_modelmesh_custom_runtime(peft_model_publish_id: str, model_name_or_path: str, namespace: str, peft_model_server_image: str):
    custom_runtime_manifest = {
        "apiVersion": "serving.kserve.io/v1alpha1",
        "kind": "ServingRuntime",
        "metadata": {
            "name": "peft-model-server",
            "namespace": namespace
        },
        "spec": {
            "supportedModelFormats": [
            {
                "name": "peft-model",
                "version": "1",
                "autoSelect": True
            }
            ],
            "multiModel": True,
            "grpcDataEndpoint": "port:8001",
            "grpcEndpoint": "port:8085",
            "containers": [
            {
                "name": "mlserver",
                "image": peft_model_server_image,
                "env": [
                {
                    "name": "MLSERVER_MODELS_DIR",
                    "value": "/models/_mlserver_models/"
                },
                {
                    "name": "MLSERVER_GRPC_PORT",
                    "value": "8001"
                },
                {
                    "name": "MLSERVER_HTTP_PORT",
                    "value": "8002"
                },
                {
                    "name": "MLSERVER_LOAD_MODELS_AT_STARTUP",
                    "value": "true"
                },
                {
                    "name": "MLSERVER_MODEL_NAME",
                    "value": "peft-model"
                },
                {
                    "name": "MLSERVER_HOST",
                    "value": "127.0.0.1"
                },
                {
                    "name": "MLSERVER_GRPC_MAX_MESSAGE_LENGTH",
                    "value": "-1"
                },
                {
                    "name": "PRETRAINED_MODEL_PATH",
                    "value": model_name_or_path
                },
                {
                    "name": "PEFT_MODEL_ID",
                    "value": peft_model_publish_id
                }
                ],
                "resources": {
                "requests": {
                    "cpu": "500m",
                    "memory": "4Gi"
                },
                "limits": {
                    "cpu": "5",
                    "memory": "5Gi"
                }
                }
            }
            ],
            "builtInAdapter": {
            "serverType": "mlserver",
            "runtimeManagementPort": 8001,
            "memBufferBytes": 134217728,
            "modelLoadingTimeoutMillis": 90000
            }
        }
    }
    custom_runtime_task = dsl.ResourceOp(
        name="deploy_modelmesh_custom_runtime",
        k8s_resource=custom_runtime_manifest,
        action="create")
    return custom_runtime_task

def serve_llm_with_peft(model_name: str, model_namespace: str):

    inference_service = '''
apiVersion: serving.kserve.io/v1beta1
kind: InferenceService
metadata:
  name: {}
  namespace: {}
  annotations:
    serving.kserve.io/deploymentMode: ModelMesh
spec:
  predictor:
    model:
      modelFormat:
        name: peft-model
      runtime: peft-model-server
      storage:
        key: localMinIO
        path: sklearn/mnist-svm.joblib
'''.format(model_name, model_namespace)

    kserve_launcher_op = comp.load_component_from_url(
        'https://raw.githubusercontent.com/kubeflow/pipelines/master/components/kserve/component.yaml')
    task = kserve_launcher_op(action="apply", inferenceservice_yaml=inference_service)
    return task


def test_modelmesh_model(modelmesh_servicename: str, modelmesh_namespace: str, model_name: str, input_tweet: str):
    import requests
    import base64
    import json

    url = "http://%s.%s:8008/v2/models/%s/infer" % (modelmesh_servicename, modelmesh_namespace, model_name)
    input_json = {
        "inputs": [
            {
            "name": "content",
            "shape": [1],
            "datatype": "BYTES",
            "data": [input_tweet]
            }
        ]
    }

    x = requests.post(url, json = input_json)

    print(x.text)
    respond_dict = json.loads(x.text)
    inference_result = respond_dict["outputs"][0]["data"][0]
    base64_bytes = inference_result.encode("ascii")
  
    string_bytes = base64.b64decode(base64_bytes)
    inference_result = string_bytes.decode("ascii")
    print("inference_result: %s " % inference_result)

test_modelmesh_model_op = comp.func_to_container_op(test_modelmesh_model, packages_to_install=['requests'], base_image='python:3.10')

# Define your pipeline function
@dsl.pipeline(
    name="Serving LLM with Prompt tuning",
    description="A Pipeline for Serving Prompt Tuning LLMs on Modelmesh"
)
def prompt_tuning_pipeline(
    peft_model_publish_id="aipipeline/bloomz-560m_PROMPT_TUNING_CAUSAL_LM",
    model_name_or_path="bigscience/bloomz-560m",
    peft_model_server_image="quay.io/aipipeline/peft-model-server:latest",
    model_name="peft-demo",
    model_namespace="modelmesh-serving",
    modelmesh_namespace = "modelmesh-serving",
    modelmesh_servicename = "modelmesh-serving",
    input_tweet = "@nationalgridus I have no water and the bill is current and paid. Can you do something about this?",
    test_served_llm_model="true"
):
    deploy_modelmesh_custom_runtime_task = deploy_modelmesh_custom_runtime(peft_model_publish_id, model_name_or_path, modelmesh_namespace, peft_model_server_image)
    serve_llm_with_peft_task = serve_llm_with_peft(model_name, model_namespace).after(deploy_modelmesh_custom_runtime_task)
    with dsl.Condition(test_served_llm_model == 'true'):
        test_modelmesh_model_task = test_modelmesh_model_op(modelmesh_servicename, modelmesh_namespace, model_name, input_tweet).after(serve_llm_with_peft_task)
        test_modelmesh_model_task.add_pod_label('pipelines.kubeflow.org/cache_enabled', 'false')

# Compile the pipeline
pipeline_func = prompt_tuning_pipeline
TektonCompiler().compile(pipeline_func, 'prompt_tuning_serving.yaml')
