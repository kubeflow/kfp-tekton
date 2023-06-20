import kfp.dsl as dsl
import kfp.components as comp
from kfp_tekton.compiler import TektonCompiler
from kfp_tekton.k8s_client_helper import env_from_secret

def prompt_tuning_bloom(peft_model_publish_id: str, model_name_or_path: str, num_epochs: int):
    from transformers import AutoModelForCausalLM, AutoTokenizer, default_data_collator, get_linear_schedule_with_warmup
    from peft import get_peft_config, get_peft_model, PromptTuningInit, PromptTuningConfig, TaskType, PeftType
    import torch
    from datasets import load_dataset
    import os
    from torch.utils.data import DataLoader
    from tqdm import tqdm

    peft_config = PromptTuningConfig(
        task_type=TaskType.CAUSAL_LM,
        prompt_tuning_init=PromptTuningInit.TEXT,
        num_virtual_tokens=8,
        prompt_tuning_init_text="Classify if the tweet is a complaint or not:",
        tokenizer_name_or_path=model_name_or_path,
    )

    dataset_name = "twitter_complaints"
    text_column = "Tweet text"
    label_column = "text_label"
    max_length = 64
    lr = 3e-2
    batch_size = 8

    dataset = load_dataset("ought/raft", dataset_name)
    dataset["train"][0]

    classes = [k.replace("_", " ") for k in dataset["train"].features["Label"].names]
    dataset = dataset.map(
        lambda x: {"text_label": [classes[label] for label in x["Label"]]},
        batched=True,
        num_proc=1,
    )

    tokenizer = AutoTokenizer.from_pretrained(model_name_or_path)
    if tokenizer.pad_token_id is None:
        tokenizer.pad_token_id = tokenizer.eos_token_id

    def preprocess_function(examples):
        batch_size = len(examples[text_column])
        inputs = [f"{text_column} : {x} Label : " for x in examples[text_column]]
        targets = [str(x) for x in examples[label_column]]
        model_inputs = tokenizer(inputs)
        labels = tokenizer(targets)
        for i in range(batch_size):
            sample_input_ids = model_inputs["input_ids"][i]
            label_input_ids = labels["input_ids"][i] + [tokenizer.pad_token_id]
            model_inputs["input_ids"][i] = sample_input_ids + label_input_ids
            labels["input_ids"][i] = [-100] * len(sample_input_ids) + label_input_ids
            model_inputs["attention_mask"][i] = [1] * len(model_inputs["input_ids"][i])
        for i in range(batch_size):
            sample_input_ids = model_inputs["input_ids"][i]
            label_input_ids = labels["input_ids"][i]
            model_inputs["input_ids"][i] = [tokenizer.pad_token_id] * (
                max_length - len(sample_input_ids)
            ) + sample_input_ids
            model_inputs["attention_mask"][i] = [0] * (max_length - len(sample_input_ids)) + model_inputs[
                "attention_mask"
            ][i]
            labels["input_ids"][i] = [-100] * (max_length - len(sample_input_ids)) + label_input_ids
            model_inputs["input_ids"][i] = torch.tensor(model_inputs["input_ids"][i][:max_length])
            model_inputs["attention_mask"][i] = torch.tensor(model_inputs["attention_mask"][i][:max_length])
            labels["input_ids"][i] = torch.tensor(labels["input_ids"][i][:max_length])
        model_inputs["labels"] = labels["input_ids"]
        return model_inputs
    
    processed_datasets = dataset.map(
        preprocess_function,
        batched=True,
        num_proc=1,
        remove_columns=dataset["train"].column_names,
        load_from_cache_file=False,
        desc="Running tokenizer on dataset",
    )

    train_dataset = processed_datasets["train"]
    eval_dataset = processed_datasets["train"]


    train_dataloader = DataLoader(
        train_dataset, shuffle=True, collate_fn=default_data_collator, batch_size=batch_size, pin_memory=False
    )
    eval_dataloader = DataLoader(eval_dataset, collate_fn=default_data_collator, batch_size=batch_size, pin_memory=False)

    model = AutoModelForCausalLM.from_pretrained(model_name_or_path)
    model = get_peft_model(model, peft_config)
    print(model.print_trainable_parameters())

    optimizer = torch.optim.AdamW(model.parameters(), lr=lr)
    lr_scheduler = get_linear_schedule_with_warmup(
        optimizer=optimizer,
        num_warmup_steps=0,
        num_training_steps=(len(train_dataloader) * num_epochs),
    )

    for epoch in range(num_epochs):
        model.train()
        total_loss = 0
        for step, batch in enumerate(tqdm(train_dataloader)):
            batch = {k: v for k, v in batch.items()}
            outputs = model(**batch)
            loss = outputs.loss
            total_loss += loss.detach().float()
            loss.backward()
            optimizer.step()
            lr_scheduler.step()
            optimizer.zero_grad()

        model.eval()
        eval_loss = 0
        eval_preds = []
        for step, batch in enumerate(tqdm(eval_dataloader)):
            batch = {k: v for k, v in batch.items()}
            with torch.no_grad():
                outputs = model(**batch)
            loss = outputs.loss
            eval_loss += loss.detach().float()
            eval_preds.extend(
                tokenizer.batch_decode(torch.argmax(outputs.logits, -1).detach().cpu().numpy(), skip_special_tokens=True)
            )

        eval_epoch_loss = eval_loss / len(eval_dataloader)
        eval_ppl = torch.exp(eval_epoch_loss)
        train_epoch_loss = total_loss / len(train_dataloader)
        train_ppl = torch.exp(train_epoch_loss)
        print("epoch=%s: train_ppl=%s train_epoch_loss=%s eval_ppl=%s eval_epoch_loss=%s" % (epoch, train_ppl, train_epoch_loss, eval_ppl, eval_epoch_loss))

    from huggingface_hub import login
    token = os.environ.get("HUGGINGFACE_TOKEN")

    login(token=token)

    peft_model_id = peft_model_publish_id
    model.push_to_hub(peft_model_id, use_auth_token=True)

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

prompt_tuning_bloom_op = comp.func_to_container_op(prompt_tuning_bloom, packages_to_install=['peft', 'transformers', 'datasets'], base_image='python:3.10')
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
    test_served_llm_model="true",
    num_epochs="50"
):
    prompt_tuning_llm = prompt_tuning_bloom_op(peft_model_publish_id, model_name_or_path, num_epochs)
    prompt_tuning_llm.add_env_variable(env_from_secret('HUGGINGFACE_TOKEN', 'huggingface-secret', 'token'))
    deploy_modelmesh_custom_runtime_task = deploy_modelmesh_custom_runtime(peft_model_publish_id, model_name_or_path, modelmesh_namespace, peft_model_server_image)
    deploy_modelmesh_custom_runtime_task.after(prompt_tuning_llm)
    serve_llm_with_peft_task = serve_llm_with_peft(model_name, model_namespace).after(deploy_modelmesh_custom_runtime_task)
    with dsl.Condition(test_served_llm_model == 'true'):
        test_modelmesh_model_task = test_modelmesh_model_op(modelmesh_servicename, modelmesh_namespace, model_name, input_tweet).after(serve_llm_with_peft_task)
        test_modelmesh_model_task.add_pod_label('pipelines.kubeflow.org/cache_enabled', 'false')

# Compile the pipeline
pipeline_func = prompt_tuning_pipeline
TektonCompiler().compile(pipeline_func, 'prompt_tuning_pipeline.yaml')
