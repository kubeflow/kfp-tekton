# objectstore

Connect and store objects in object store. 

**Purpose**: use object store for storing the results of custom tasks.

**For example:**

1. To setup connection.
```go
import 	"github.com/kubeflow/kfp-tekton/tekton-catalog/objectstore/pkg/writer"

func init() {
	config := writer.ObjectStoreWriterConfig{
		CreateBucket:      true,
		DefaultBucketName: "testing-bucket",
		AccessKey:         "<>",
		SecretKey:         "<>",
		Region:            "us-south",
		ServiceEndpoint:   "https://s3.us-south.cloud-object-storage.appdomain.cloud",
		Token:             "",
		S3ForcePathStyle:  false,
	}
	w = writer.Writer{}
	err := w.Load(config)
	if err != nil {
		fmt.Printf("error while connecting to s3 %#v \n", err)
	} else {
		fmt.Printf("\nConnected to s3.\n")
	}
}
```

2. Store results of tasks.

```go
result := fmt.Sprintf("%s", out.ConvertToType(types.StringType).Value())
        // retrieve Pipeline Run name and task name as follows.
		prName := run.ObjectMeta.Labels["tekton.dev/pipelineRun"]
		taskName:= run.ObjectMeta.Labels["tekton.dev/pipelineTask"]
		err = w.Write(prName, taskName, param.Name, []byte(result))
		if err != nil {
			logger.Errorf("error while writing to s3 %#v", err)
		}
		runResults = append(runResults, v1alpha1.RunResult{
			Name:  param.Name,
			Value: result,
		})
```

__Results are stored as:__

`/artifacts/$PIPELINERUN/$PIPELINETASK/<tekton_result_name>.tgz`
