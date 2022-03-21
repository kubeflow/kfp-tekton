# objectstore

## Connect and store objects in object store. 

**Purpose**: use object store for storing the results of custom tasks.

**For example:**

1. To setup connection.
```go
import 	"github.com/kubeflow/kfp-tekton/tekton-catalog/objectstore/pkg/writer"

func init() {
	config := writer.ObjectStoreConfig{
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

## Setup log to object store for golang.

Log to cloud object storage for golang implemented as `io.Writer`.

### Use it as a plugin/extension to [uber-go/zap](https://github.com/uber-go/zap) logger

Configure logger and add a multi write syncer for `zap` as follows,

```go
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	cl "github.com/kubeflow/kfp-tekton/tekton-catalog/objectstore/pkg/writer"
)

func initializeLogger() {

	loggerConfig := cl.ObjectStoreConfig{}
	objectStoreLogger := cl.Logger{
		MaxSize: 1024 * 100, // After reaching this size the buffer syncs with object store.
	}
    
	// Provide all the configuration.
	loggerConfig.Enable = true
	loggerConfig.AccessKey = "key"
	loggerConfig.SecretKey = "key_secret"
	loggerConfig.Region = "us-south"
	loggerConfig.ServiceEndpoint = "<url>"
	loggerConfig.DefaultBucketName = "<bucket_name>"
	loggerConfig.CreateBucket = false // If the bucket already exists.

	_ = objectStoreLogger.LoadDefaults(loggerConfig)
	// setup a multi sync Writer as follows,
	w := zapcore.NewMultiWriteSyncer(
		zapcore.AddSync(os.Stdout),
		zapcore.AddSync(&objectStoreLogger),
	)
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		w,
		zap.InfoLevel,
	)
	logger := zap.New(core)
	logger.Info("First log msg with object store logger.")
}

// If you wish to sync with object store before shutdown.
func shutdownHook() {
	// set up SIGHUP to send logs to object store before shutdown.
	signal.Ignore(syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
	c := make(chan os.Signal, 3)
	signal.Notify(c, syscall.SIGTERM)
	signal.Notify(c, syscall.SIGINT)
	signal.Notify(c, syscall.SIGHUP)

	go func() {
		for {
			<-c
			err := objectStoreLogger.Close() //causes a sync with object store.
			fmt.Printf("Synced with object store... %v", err)
			os.Exit(0)
		}
	}()
}

```