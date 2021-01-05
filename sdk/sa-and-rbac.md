# Permission issues

When running some kfp-tekton pipelines on your cluster, you may encounter permission issues
when the Tekton pipeline pods try to create a resource. For example to run
[resourceop_basic](/sdk/python/tests/compiler/testdata/resourceop_basic.py), you may encounter
this error:

```bash
tkn pipeline start resourceop-basic --showlog
Pipelinerun started: resourceop-basic-run-w54mv
Waiting for logs to be available...
[test-step : test-step] time="2020-04-15T22:32:50Z" level=error msg="Initialize script failed: exit status 2:"
[test-step : test-step] time="2020-04-15T22:32:50Z" level=info msg="kubectl create -f /tmp/manifest.yaml -o json"
[test-step : test-step] time="2020-04-15T22:32:51Z" level=error msg="Run kubectl command failed with: exit status 1 and Error from server (Forbidden): error when creating \"/tmp/manifest.yaml\": jobs.batch is forbidden: User \"system:serviceaccount:tekton-pipelines:default\" cannot create resource \"jobs\" in API group \"batch\" in the namespace \"tekton-pipelines\""
[test-step : test-step] time="2020-04-15T22:32:51Z" level=error msg="Execute resource failed: exit status 1:"

failed to get logs for task test-step : container step-test-step has failed  : [{"key":"StartedAt","value":"2020-04-15T22:32:50Z","resourceRef":{}}]
```

In the above case, `tekton-pipelines` is using `default` ServiceAccount which doesn't have
[RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/) setup, or the `RBAC`
doesn't have enough permission for the pod to create resource.

There are two ways to solve this. We list the details here.

## Setup RBAC with default serviceAccount 

Create a ClusterRoleBinding with `cluster-admin` to the default service account.
In this case, the default service account is `default` and we are deploying it to
the `tekton-pipelines` namespace.

Create `RBAC` permission as below:

```bash
cat <<EOF |kubectl apply -f -
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: default-admin
subjects:
  - kind: ServiceAccount
    name: default
    namespace: tekton-pipelines
roleRef:
  kind: ClusterRole
  name: cluster-admin
  apiGroup: rbac.authorization.k8s.io
EOF
```
After seeing this message:

```bash
clusterrolebinding.rbac.authorization.k8s.io/default-admin created
```
Re-run the Tekton pipeline

```bash
tkn pipeline start resourceop-basic --showlog
Pipelinerun started: resourceop-basic-run-lxb6d
Waiting for logs to be available...
[test-step : test-step] time="2020-04-16T04:44:33Z" level=error msg="Initialize script failed: exit status 2:"
[test-step : test-step] time="2020-04-16T04:44:33Z" level=info msg="kubectl create -f /tmp/manifest.yaml -o json"
[test-step : test-step] time="2020-04-16T04:44:34Z" level=info msg=tekton-pipelines/Job.batch/resourceop-basic-job-q9lw5
[test-step : test-step] time="2020-04-16T04:44:34Z" level=info msg="Saving resource output parameters"
[test-step : test-step] time="2020-04-16T04:44:34Z" level=info msg="[kubectl get Job.batch/resourceop-basic-job-q9lw5 -o jsonpath={.metadata.name} -n tekton-pipelines]"
[test-step : test-step] time="2020-04-16T04:44:34Z" level=info msg="Saved output parameter: {.metadata.name}, value: resourceop-basic-job-q9lw5"
[test-step : test-step] time="2020-04-16T04:44:34Z" level=info msg="[kubectl get Job.batch/resourceop-basic-job-q9lw5 -o jsonpath={} -n tekton-pipelines]"
[test-step : test-step] time="2020-04-16T04:44:35Z" level=info msg="Saved output parameter: {}, value: map[apiVersion:batch/v1 kind:Job metadata:map[creationTimestamp:2020-04-16T04:44:31Z generateName:resourceop-basic-job- labels:map[controller-uid:a7e443db-361d-4e71-925c-fd8664017359 job-name:resourceop-basic-job-q9lw5] name:resourceop-basic-job-q9lw5 namespace:tekton-pipelines resourceVersion:9922678 selfLink:/apis/batch/v1/namespaces/tekton-pipelines/jobs/resourceop-basic-job-q9lw5 uid:a7e443db-361d-4e71-925c-fd8664017359] spec:map[backoffLimit:4 completions:1 parallelism:1 selector:map[matchLabels:map[controller-uid:a7e443db-361d-4e71-925c-fd8664017359]] template:map[metadata:map[creationTimestamp:<nil> labels:map[controller-uid:a7e443db-361d-4e71-925c-fd8664017359 job-name:resourceop-basic-job-q9lw5] name:resource-basic] spec:map[containers:[map[command:[/usr/bin/env] image:k8s.gcr.io/busybox imagePullPolicy:Always name:sample-container resources:map[] terminationMessagePath:/dev/termination-log terminationMessagePolicy:File]] dnsPolicy:ClusterFirst restartPolicy:Never schedulerName:default-scheduler securityContext:map[] terminationGracePeriodSeconds:30]]] status:map[active:1 startTime:2020-04-16T04:44:31Z]]"
```

## Setup RBAC with customized ServiceAccount

If you want to use customized ServiceAccount, you can bind it with RBAC. For example,
if you have a ServiceAccount name: `test-sa-rbac` in the `tekton-pipelines` namespace,
create `RBAC` permission as defined below for that ServiceAccount:

```bash
cat <<EOF |kubectl apply -f -
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: default-admin
subjects:
  - kind: ServiceAccount
    name: test-sa-rbac
    namespace: tekton-pipelines
roleRef:
  kind: ClusterRole
  name: cluster-admin
  apiGroup: rbac.authorization.k8s.io
EOF
```
After you see this message
```bash
clusterrolebinding.rbac.authorization.k8s.io/default-admin configured
```

Use this tekton cli command to run the pipeline with your ServiceAccount

```bash
tkn pipeline start resourceop-basic -s test-sa-rbac --showlog
```

Here is how the output looks like:

```bash
Pipelinerun started: resourceop-basic-run-xnlls
Waiting for logs to be available...
[test-step : test-step] time="2020-04-16T05:27:24Z" level=error msg="Initialize script failed: exit status 2:"
[test-step : test-step] time="2020-04-16T05:27:24Z" level=info msg="kubectl create -f /tmp/manifest.yaml -o json"
[test-step : test-step] time="2020-04-16T05:27:25Z" level=info msg=tekton-pipelines/Job.batch/resourceop-basic-job-qwxhs
[test-step : test-step] time="2020-04-16T05:27:25Z" level=info msg="Saving resource output parameters"
[test-step : test-step] time="2020-04-16T05:27:25Z" level=info msg="[kubectl get Job.batch/resourceop-basic-job-qwxhs -o jsonpath={.metadata.name} -n tekton-pipelines]"
[test-step : test-step] time="2020-04-16T05:27:25Z" level=info msg="Saved output parameter: {.metadata.name}, value: resourceop-basic-job-qwxhs"
[test-step : test-step] time="2020-04-16T05:27:25Z" level=info msg="[kubectl get Job.batch/resourceop-basic-job-qwxhs -o jsonpath={} -n tekton-pipelines]"
[test-step : test-step] time="2020-04-16T05:27:25Z" level=info msg="Saved output parameter: {}, value: map[apiVersion:batch/v1 kind:Job metadata:map[creationTimestamp:2020-04-16T05:27:22Z generateName:resourceop-basic-job- labels:map[controller-uid:7e52145e-a9bc-45a3-9516-21105963a3dc job-name:resourceop-basic-job-qwxhs] name:resourceop-basic-job-qwxhs namespace:tekton-pipelines resourceVersion:9933832 selfLink:/apis/batch/v1/namespaces/tekton-pipelines/jobs/resourceop-basic-job-qwxhs uid:7e52145e-a9bc-45a3-9516-21105963a3dc] spec:map[backoffLimit:4 completions:1 parallelism:1 selector:map[matchLabels:map[controller-uid:7e52145e-a9bc-45a3-9516-21105963a3dc]] template:map[metadata:map[creationTimestamp:<nil> labels:map[controller-uid:7e52145e-a9bc-45a3-9516-21105963a3dc job-name:resourceop-basic-job-qwxhs] name:resource-basic] spec:map[containers:[map[command:[/usr/bin/env] image:k8s.gcr.io/busybox imagePullPolicy:Always name:sample-container resources:map[] terminationMessagePath:/dev/termination-log terminationMessagePolicy:File]] dnsPolicy:ClusterFirst restartPolicy:Never schedulerName:default-scheduler securityContext:map[] terminationGracePeriodSeconds:30]]] status:map[active:1 startTime:2020-04-16T05:27:22Z]]"
```

For more information about the Tekton CLI command, take a look at the
[tkn pipeline start](https://github.com/tektoncd/cli/blob/master/docs/cmd/tkn_pipeline_start.md)
document.
