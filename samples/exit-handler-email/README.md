# Email notification via SMTP server

This pipeline demonstrates how to use the exit handler in the Kubeflow pipeline to send email notifications via the SMTP server. The exit component is based on the [send-email](https://github.com/tektoncd/catalog/tree/master/task/sendmail/0.1) Tekton catalog task.

## prerequisites
- Install [KFP Tekton prerequisites](/samples/README.md)
- Host or subscribe to a [SMTP server](https://en.wikipedia.org/wiki/Simple_Mail_Transfer_Protocol)

## Instructions
1. Modify [secret.yaml](secret.yaml) with the following information. Then create the secret under the Kubeflow namespace (for single user) or User namespace (for multi-user).

* **url**: The IP address of the SMTP server

* **port**: The port number of the SMTP server

* **user**: User name for the SMTP server

* **password**: Password for the SMTP server

* **tls**: The tls enabled or not ("True" or "False")

    ```shell
    NAMESPACE=kubeflow
    kubectl apply -f secret.yaml -n ${NAMESPACE}
    ```

2. Create a shared persistent volume claim for passing optional attachment.

    ```shell
    kubectl apply -f pvc.yaml -n ${NAMESPACE}
    ```

3. Compile the send-email pipeline using the compiler inside the python code. The kfp-tekton SDK will produce a Tekton pipeline yaml definition in the same directory called `email_pipeline.yaml`.
    ```shell
    # Compile the python code
    python send-email.py
    ```

Then, upload the `email_pipeline.yaml` file to the Kubeflow pipeline dashboard with Tekton Backend to run this pipeline.

### Pipeline parameters

* **server**: The name of the secret that has the SMTP server information

* **subject**: Email subject (plain text)

* **body**: Email body (plain text)

* **sender**: Email sender email address

* **recipients**: Email recipients email addresses (comma space delimited)

* **attachment_path**: Optional attachment path from the previous path
