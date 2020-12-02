import json
from string import Template

import kfp
from kfp import components
from kfp.components import func_to_container_op
import kfp.dsl as dsl

# ibmc-file-gold is the recommended ReadWriteMany storageclass for IBM Cloud.
storageclass = 'ibmc-file-gold'
model_name = "mnist-demo"
user_namespace = "anonymous"

def convert_mnist_experiment_result(experiment_result) -> str:
    import json
    r = json.loads(experiment_result)
    args = []
    for hp in r:
        print(hp)
        args.append("%s=%s" % (hp["name"], hp["value"]))

    return " ".join(args)

def add_istio_annotation(op):
    op.add_pod_annotation(name='sidecar.istio.io/inject', value='false')
    return op

@dsl.pipeline(
    name="End to end pipeline",
    description="An end to end example including hyperparameter tuning, train and inference."
)
def mnist_pipeline(
        name=model_name,
        namespace=user_namespace,
        storageclass=storageclass,
        step=4000):
    # step 1: create a Katib experiment to tune hyperparameters
    objectiveConfig = {
      "type": "minimize",
      "goal": 0.001,
      "objectiveMetricName": "loss",
    }
    algorithmConfig = {"algorithmName" : "random"}
    parameters = [
      {"name": "--tf-learning-rate", "parameterType": "double", "feasibleSpace": {"min": "0.01","max": "0.03"}},
      {"name": "--tf-batch-size", "parameterType": "discrete", "feasibleSpace": {"list": ["16", "32", "64"]}},
    ]
    rawTemplate = {
      "apiVersion": "kubeflow.org/v1",
      "kind": "TFJob",
      "metadata": {
         "name": "{{.Trial}}",
         "namespace": "{{.NameSpace}}"
      },
      "spec": {
        "tfReplicaSpecs": {
          "Chief": {
            "replicas": 1,
            "restartPolicy": "OnFailure",
            "template": {
              "spec": {
                "containers": [
                {
                  "command": [
                    "sh",
                    "-c"
                  ],
                  "args": [
                    "python /opt/model.py --tf-train-steps=2000 {{- with .HyperParameters}} {{- range .}} {{.Name}}={{.Value}} {{- end}} {{- end}}"
                  ],
                  "image": "liuhougangxa/tf-estimator-mnist",
                  "name": "tensorflow"
                }
                ]
              }
            }
          },
          "Worker": {
            "replicas": 3,
            "restartPolicy": "OnFailure",
            "template": {
              "spec": {
                "containers": [
                {
                  "command": [
                    "sh",
                    "-c"
                  ],
                  "args": [ 
                    "python /opt/model.py --tf-train-steps=2000 {{- with .HyperParameters}} {{- range .}} {{.Name}}={{.Value}} {{- end}} {{- end}}"
                  ],
                  "image": "liuhougangxa/tf-estimator-mnist",
                  "name": "tensorflow"
                }
                ]
              }
            }
          }
        }
      }
    }
    
    trialTemplate = {
      "goTemplate": {
        "rawTemplate": json.dumps(rawTemplate)
      }
    }

    metricsCollectorSpec = {
      "source": {
        "fileSystemPath": {
          "path": "/tmp/tf",
          "kind": "Directory"
        }
      },
      "collector": {
        "kind": "TensorFlowEvent"
      }
    }

    katib_experiment_launcher_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/master/components/kubeflow/katib-launcher/component.yaml')
    op1 = katib_experiment_launcher_op(
            experiment_name=name,
            experiment_namespace=namespace,
            parallel_trial_count=3,
            max_trial_count=12,
            objective=str(objectiveConfig),
            algorithm=str(algorithmConfig),
            trial_template=str(trialTemplate),
            parameters=str(parameters),
            metrics_collector=str(metricsCollectorSpec),
            # experiment_timeout_minutes=experimentTimeoutMinutes,
            delete_finished_experiment=False)

    # step2: create a TFJob to train your model with best hyperparameter tuned by Katib
    tfjobjson_template = Template("""
{
  "apiVersion": "kubeflow.org/v1",
  "kind": "TFJob",
  "metadata": {
    "name": "$name",
    "namespace": "$namespace",
    "annotations": {
        "sidecar.istio.io/inject": "false"
    }
  },
  "spec": {
    "tfReplicaSpecs": {
      "Chief": {
        "replicas": 1,
        "restartPolicy": "OnFailure",
        "template": {
          "metadata": {
            "annotations": {
              "sidecar.istio.io/inject": "false"
            }
          },
          "spec": {
            "volumes": [
              {
                "name": "export-model",
                "persistentVolumeClaim": {
                  "claimName": "$modelpvc"
                }
              }
            ],
            "containers": [
              {
                "command": [
                  "sh",
                  "-c"
                ],
                "args": [
                  "python /opt/model.py --tf-train-steps=$step --tf-export-dir=/mnt/export $args"
                ],
                "image": "liuhougangxa/tf-estimator-mnist",
                "name": "tensorflow",
                "volumeMounts": [
                  {
                    "mountPath": "/mnt/export",
                    "name": "export-model"
                  }
                ]
              }
            ]
          }
        }
      },
      "Worker": {
        "replicas": 3,
        "restartPolicy": "OnFailure",
        "template": {
          "metadata": {
            "annotations": {
              "sidecar.istio.io/inject": "false"
            }
          },
          "spec": {
            "volumes": [
              {
                "name": "export-model",
                "persistentVolumeClaim": {
                  "claimName": "$modelpvc"
                }
              }
            ],
            "containers": [
              {
                "command": [
                  "sh",
                  "-c"
                ],
                "args": [
                  "python /opt/model.py --tf-train-steps=$step --tf-export-dir=/mnt/export $args"
                ],
                "image": "liuhougangxa/tf-estimator-mnist",
                "name": "tensorflow",
                "volumeMounts": [
                  {
                    "mountPath": "/mnt/export",
                    "name": "export-model"
                  }
                ]
              }
            ]
          }
        }
      }
    }
  }
}
""")

    convert_op = func_to_container_op(convert_mnist_experiment_result)
    op2 = convert_op(op1.output)
    
    volume_template = Template("""
{
  "apiVersion": "v1",
  "kind": "PersistentVolumeClaim",
  "metadata": {
    "name": "{{workflow.name}}-modelpvc",
    "namespace": "$namespace"
  },
  "spec": {
      "accessModes": ["ReadWriteMany"],
      "resources": {
          "requests": {
              "storage": "1Gi"
          }
      },
      "storageClassName": "$storageclass"
   }
}
""")
    
    volopjson = volume_template.substitute({'namespace': namespace, 'storageclass': storageclass})
    volop = json.loads(volopjson)

    modelvolop = dsl.ResourceOp(
        name="modelpvc",
        k8s_resource=volop
    )

    tfjobjson = tfjobjson_template.substitute(
            {'args': op2.output,
             'name': name,
             'namespace': namespace,
             'step': step,
             'modelpvc': modelvolop.outputs["name"]
            })

    tfjob = json.loads(tfjobjson)

    train = dsl.ResourceOp(
        name="train",
        k8s_resource=tfjob,
        success_condition='status.replicaStatuses.Worker.succeeded==3,status.replicaStatuses.Chief.succeeded==1'
    )

    # step 3: model inferencese by KFServing Inferenceservice
    inferenceservice_template = Template("""
{
  "apiVersion": "serving.kubeflow.org/v1alpha2",
  "kind": "InferenceService",
  "metadata": {
    "name": "$name",
    "namespace": "$namespace"
  },
  "spec": {
    "default": {
      "predictor": {
        "tensorflow": {
          "storageUri": "pvc://$modelpvc/"
        }
      }
    }
  }
}
""")
    inferenceservicejson = inferenceservice_template.substitute({'modelpvc': modelvolop.outputs["name"],
                                                                 'name': name,
                                                                 'namespace': namespace})
    inferenceservice =  json.loads(inferenceservicejson)
    inference = dsl.ResourceOp(
      name="inference",
      k8s_resource=inferenceservice,
      success_condition='status.url').after(train)
    
    dsl.get_pipeline_conf().add_op_transformer(add_istio_annotation)


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(mnist_pipeline, __file__.replace('.py', '.yaml'))
