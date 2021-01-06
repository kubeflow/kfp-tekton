## Any Sequencer

The `Any Sequencer` is going to handle the requirement: If Job4 depends on Job1, Job2 and Job3, when any one of the Job1, job2 or job3 complete successfully, Job4 will be started. Order doesn’t matter, but the `Any Sequencer` don’t wait for all the job status.

Implement an `taskRun` status watcher task to watch the list of `taskRun` it depends on (using kubenertes RBAC), if one or more is completed, we exit the watching task and continue our workload. This status watcher task can be implemented in the same way as our “condition” task to make it “dependable”. 

Note that the service account of the `Any Sequencer` need get premission to watch the status of sepecified `taskRun`.

### How to build binary

```
make build-linux
```

### How to build image

```
make image
```
