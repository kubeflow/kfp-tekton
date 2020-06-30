import json
from kfp import components
import kfp.dsl as dsl


katib_experiment_launcher_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/master/components/kubeflow/katib-launcher/component.yaml')
fairness_check_ops = components.load_component_from_url('https://raw.githubusercontent.com/IBM/AIF360/master/mlops/kubeflow/bias_detector_pytorch/component.yaml')
robustness_check_ops = components.load_component_from_url('https://raw.githubusercontent.com/IBM/adversarial-robustness-toolbox/master/utils/mlops/kubeflow/robustness_evaluation_fgsm_pytorch/component.yaml')


@dsl.pipeline(
    name="Launch trusted-ai pipeline",
    description="An example for trusted-ai integration."
)
def trusted_ai(
        name="trusted-ai",
        namespace="anonymous",
        goal="0.99",
        parallelTrialCount="1",
        maxTrialCount="1",
        experimentTimeoutMinutes="60",
        deleteAfterDone="True",
        fgsm_attack_epsilon='0.2',
        model_class_file='PyTorchModel.py',
        model_class_name='ThreeLayerCNN',
        feature_testset_path='processed_data/X_test.npy',
        label_testset_path='processed_data/y_test.npy',
        protected_label_testset_path='processed_data/p_test.npy',
        favorable_label='0.0',
        unfavorable_label='1.0',
        privileged_groups="[{'race': 0.0}]",
        unprivileged_groups="[{'race': 4.0}]",
        loss_fn='torch.nn.CrossEntropyLoss()',
        optimizer='torch.optim.Adam(model.parameters(), lr=0.001)',
        clip_values='(0, 1)',
        nb_classes='2',
        input_shape='(1,3,64,64)'):
    objectiveConfig = {
      "type": "maximize",
      "goal": goal,
      "objectiveMetricName": "accuracy",
      "additionalMetricNames": []
    }
    algorithmConfig = {"algorithmName" : "random"}
    parameters = [
      {"name": "--dummy", "parameterType": "int", "feasibleSpace": {"min": "1", "max": "2"}},
    ]
    rawTemplate = {
      "apiVersion": "batch/v1",
      "kind": "Job",
      "metadata": {
         "name": "{{.Trial}}",
         "namespace": "{{.NameSpace}}"
      },
      "spec": {
        "template": {
          "spec": {
            "restartPolicy": "Never",
            "containers": [
              {"name": "{{.Trial}}",
               "image": "aipipeline/gender-classification:latest",
               "command": [
                   "python", "-u", "gender_classification_training.py", "--data_bucket", "mlpipeline", "--result_bucket", "mlpipeline"
               ]
              }
            ],
            "env": [{'name': 'S3_ENDPOINT', 'value': 'minio-service.kubeflow:9000'}]
          }
        }
      }
    }
    trialTemplate = {
      "goTemplate": {
        "rawTemplate": json.dumps(rawTemplate)
      }
    }
    katib_run = katib_experiment_launcher_op(
        experiment_name=name,
        experiment_namespace=namespace,
        parallel_trial_count=parallelTrialCount,
        max_trial_count=maxTrialCount,
        objective=str(objectiveConfig),
        algorithm=str(algorithmConfig),
        trial_template=str(trialTemplate),
        parameters=str(parameters),
        experiment_timeout_minutes=experimentTimeoutMinutes,
        delete_finished_experiment=deleteAfterDone)

    fairness_check = fairness_check_ops(model_id='training-example',
                                        model_class_file=model_class_file,
                                        model_class_name=model_class_name,
                                        feature_testset_path=feature_testset_path,
                                        label_testset_path=label_testset_path,
                                        protected_label_testset_path=protected_label_testset_path,
                                        favorable_label=favorable_label,
                                        unfavorable_label=unfavorable_label,
                                        privileged_groups=privileged_groups,
                                        unprivileged_groups=unprivileged_groups,
                                        data_bucket_name='mlpipeline',
                                        result_bucket_name='mlpipeline').after(katib_run).set_image_pull_policy("Always")
    robustness_check = robustness_check_ops(model_id='training-example',
                                            epsilon=fgsm_attack_epsilon,
                                            model_class_file=model_class_file,
                                            model_class_name=model_class_name,
                                            feature_testset_path=feature_testset_path,
                                            label_testset_path=label_testset_path,
                                            loss_fn=loss_fn,
                                            optimizer=optimizer,
                                            clip_values=clip_values,
                                            nb_classes=nb_classes,
                                            input_shape=input_shape,
                                            data_bucket_name='mlpipeline',
                                            result_bucket_name='mlpipeline').after(katib_run).set_image_pull_policy("Always")

if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(trusted_ai, __file__.replace('.py', '.yaml'))
