# TFX On-Prem Demo

This Demo is based on the [TFX taxi oss example](https://github.com/kubeflow/pipelines/blob/master/samples/core/parameterized_tfx_oss/parameterized_tfx_oss.py) and designed to run the TFX Chicago Taxi example on-prem without any GCS/GCP dependency.

## Prerequisites
1. Provision a Kubernetes or OpenShift cluster.
2. Install [KubeFlow](https://www.kubeflow.org/docs/started/getting-started/)
3. Install KubeFlow Pipeline and TensorFlow Extended Python libraries
   ```shell
   pip install tfx kfp --upgrade
   ```


## Instructions
1. Create the Persistent Volume Claim as our on-prem shared storage for the TFX example.
   > If the Kubernetes/OpenShift cluster doesn't have a default storageclass, then we need to create a persistent volume using `kubectl apply -f pvc/pv.yaml`

   Create PVC:
   ```shell
   kubectl apply -f pvc/tfx-pvc.yaml
   ```

2. Deploy filebrowser to move files into the PVC storage.
   ```shell
   kubectl apply -f pvc/filebrowser.yaml
   kubectl get svc -n kubeflow | grep filebrowser
   ```

   Then navigate to the filebrowser webpage and verify it's deployed. The endpoint should be the <cluster_node_ip>:<filebrowser_nodeport>

3. Move the example dataset and util files into the PVC. First we need to download the dataset and util files from Google Cloud Storage.
   ```shell
   curl https://storage.googleapis.com/ml-pipeline-playground/tfx_taxi_simple/modules/taxi_utils.py -o taxi_utils.py
   curl https://storage.googleapis.com/ml-pipeline-playground/tfx_taxi_simple/data/data.csv -o data.csv
   ```

   Then, upload the `taxi_utils.py` under `modules/taxi_utils.py` and `data.csv` under `data/data.csv` in the filebrowser GUI.

4. Compile the TFX TAXI On-Prem example.
   ```shell
   python TFX-taxi-sample-pvc.py
   ```

   Then upload the `chicago_taxi_pipeline_simple.tar.gz` on the KubeFlow UI and run it.

5. To serve the model with KFServing, upload the `tfx-kfs.yaml` on the KubeFlow UI and run it.
