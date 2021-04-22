## Any Sequencer

`Any Sequencer`: When any one of the task dependencies complete successfully, the dependent task will be started. Order of execution of the dependencies doesn’t matter, e.g. if Job4 depends on Job1, Job2 and Job3, and when any one of the Job1, Job2 or Job3 complete successfully, Job4 will be started. Order doesn’t matter, and `Any Sequencer` doesn’t wait for all the job completions.

This implements a `taskRun` status watcher task to watch the list of `taskRun` it depends on (using Kubernetes RBAC). If one or more is completed, it exits the watching task and continues moving forward with Pipeline run. This status watcher task can be implemented in the same way as our “condition” task to make it “dependable”. 

Note that the service account of `Any Sequencer` needs `get` permission to watch the status of the specified `taskRun`.
 
### How to build binary

```
make build-linux
```

### How to build image

```
make image
```
