/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pipelinelooprun

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/IBM/ibm-cos-sdk-go/aws"
	"github.com/IBM/ibm-cos-sdk-go/aws/credentials"
	"github.com/IBM/ibm-cos-sdk-go/aws/credentials/ibmiam"
	"github.com/IBM/ibm-cos-sdk-go/aws/session"
	"github.com/IBM/ibm-cos-sdk-go/service/s3"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/system"
)

type ObjectStoreLogConfig struct {
	Enable              bool
	defaultBucketName   string
	accessKey           string
	secretKey           string
	ibmStyleCredentials bool
	apiKey              string
	serviceInstanceID   string
	region              string
	serviceEndpoint     string
	authEndpoint        string
	client              *s3.S3
}

type Logger struct {
	buffer *bytes.Buffer
	// When buffer reaches the size of MaxSize, it tries to sync with object store.
	MaxSize int64
	// Whether to compress before syncing the buffer.
	Compress bool
	// Current size of the buffer.
	size int64
	// Sync irrespective of buffer size after elapsing this interval.
	SyncInterval time.Duration
	mu           sync.Mutex
	LogConfig    *ObjectStoreLogConfig
}

// ensure we always implement io.WriteCloser
var _ io.WriteCloser = (*Logger)(nil)

func (l *Logger) Write(p []byte) (n int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	writeLen := int64(len(p))
	if l.size+writeLen >= l.MaxSize {
		if err := l.syncBuffer(); err != nil {
			return 0, err
		}
	}
	if n, err = l.buffer.Write(p); err != nil {
		return n, err
	}
	l.size = l.size + int64(n)
	return n, nil
}

func (l *Logger) syncBuffer() error {
	fmt.Printf("Syncing buffer size : %d, MaxSize: %d \n", l.size, l.MaxSize)
	err := l.LogConfig.writeToObjectStore(l.LogConfig.defaultBucketName,
		time.Now().Format(time.RFC3339Nano), l.buffer.Bytes())
	if err != nil {
		return err
	}
	l.buffer.Reset()
	l.size = 0
	return nil
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.syncBuffer()
}

func (o *ObjectStoreLogConfig) load(ctx context.Context, kubeClientSet kubernetes.Interface) error {
	configMap, err := kubeClientSet.CoreV1().ConfigMaps(system.Namespace()).
		Get(ctx, "object-store-config", metaV1.GetOptions{})
	if err != nil {
		return err
	}
	if o.Enable, err = strconv.ParseBool(configMap.Data["enable"]); err != nil || !o.Enable {
		return err
	}

	if o.ibmStyleCredentials, err = strconv.ParseBool(configMap.Data["ibmStyleCredentials"]); err != nil {
		return err
	}

	o.apiKey = configMap.Data["apiKey"]
	o.accessKey = configMap.Data["accessKey"]
	o.secretKey = configMap.Data["secretKey"]
	o.serviceInstanceID = configMap.Data["serviceInstanceID"]
	o.region = configMap.Data["region"]
	o.serviceEndpoint = configMap.Data["serviceEndpoint"]
	o.authEndpoint = configMap.Data["authEndpoint"]
	o.defaultBucketName = configMap.Data["defaultBucketName"]
	ibmCredentials := ibmiam.NewStaticCredentials(aws.NewConfig(), o.authEndpoint, o.apiKey, o.serviceInstanceID)
	s3Credentials := credentials.NewStaticCredentials(o.accessKey, o.secretKey, "")
	var creds *credentials.Credentials
	if o.ibmStyleCredentials {
		creds = ibmCredentials
	} else {
		creds = s3Credentials
	}
	// Create client config
	var conf = aws.NewConfig().
		WithRegion(o.region).
		WithEndpoint(o.serviceEndpoint).
		WithCredentials(creds).
		WithS3ForcePathStyle(true)

	var sess = session.Must(session.NewSession())
	o.client = s3.New(sess, conf)
	input := &s3.CreateBucketInput{
		Bucket: aws.String(o.defaultBucketName),
	}
	_, err = o.client.CreateBucket(input)
	if err != nil {
		fmt.Printf("This error might be harmless, as the default bucket may already exist, %v\n",
			err.Error())
	}
	return nil
}

func (o *ObjectStoreLogConfig) CreateNewBucket(bucketName string) error {
	if !o.Enable || bucketName == o.defaultBucketName {
		return nil
	}
	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}
	_, err := o.client.CreateBucket(input)
	return err
}

func (o *ObjectStoreLogConfig) writeToObjectStore(bucketName string, key string, content []byte) error {
	if !o.Enable {
		return nil
	}
	input := s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
		Body:   bytes.NewReader(content),
	}

	_, err := o.client.PutObject(&input)
	// fmt.Printf("Response from object store: %v\n", obj)
	return err
}

func (l *Logger) LoadDefaults(ctx context.Context, kubeClientSet kubernetes.Interface) error {

	if l.LogConfig == nil {
		l.LogConfig = &ObjectStoreLogConfig{}
		err := l.LogConfig.load(ctx, kubeClientSet)
		if err != nil {
			return err
		}
		if !l.LogConfig.Enable {
			return fmt.Errorf("Object store logging is disabled. " +
				"Please edit `object-store-config` configMap to setup logging.\n")
		}
	}
	if l.buffer == nil {
		l.buffer = new(bytes.Buffer)
	}
	return nil
}
