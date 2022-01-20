# cos-logger

Log to cloud object storage for golang implemented as `io.Writer`.

## Use it as a plugin/extension to [uber-go/zap](https://github.com/uber-go/zap) logger

Configure logger and add a multi write syncer for `zap` as follows,

```go
package cos_logger
import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
    cl "github.com/scrapcodes/cos-logger"
)

func initializeLogger() {

	loggerConfig := cl.ObjectStoreLogConfig{}
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
	// setup a multi sync writer as follows,
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
			err := objectStoreLogger.Close()
			fmt.Printf("Synced with object store... %v", err)
			os.Exit(0)
		}
	}()
}

```