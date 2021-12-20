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
	"strconv"
	"time"

	"github.com/IBM/ibm-cos-sdk-go/aws"
	"github.com/IBM/ibm-cos-sdk-go/aws/credentials"
	"github.com/IBM/ibm-cos-sdk-go/aws/credentials/ibmiam"
	"github.com/IBM/ibm-cos-sdk-go/aws/session"
	"github.com/IBM/ibm-cos-sdk-go/service/s3"
	"go.uber.org/zap"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/system"
)

type ObjectStoreConfig struct {
	enable              bool
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
	logger              *zap.SugaredLogger
}

func (o *ObjectStoreConfig) load(ctx context.Context, kubeClientSet kubernetes.Interface) error {
	o.logger = logging.FromContext(ctx)

	configMap, err := kubeClientSet.CoreV1().ConfigMaps(system.Namespace()).
		Get(ctx, "object-store-config", metaV1.GetOptions{})
	if err != nil {
		return err
	}
	if o.enable, err = strconv.ParseBool(configMap.Data["enable"]); err != nil || !o.enable {
		o.logger.Infof("Log to object store is disabled. %v", err)
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
		o.logger.Error(err)
	}
	return nil
}

func (o *ObjectStoreConfig) CreateNewBucket(bucketName string) error {
	if !o.enable || bucketName == o.defaultBucketName {
		return nil
	}
	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}
	_, err := o.client.CreateBucket(input)
	return err
}

func (o *ObjectStoreConfig) writeToObjectStore(bucketName string, key string, content string) error {
	if !o.enable {
		return nil
	}
	input := s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
		Body:   bytes.NewReader([]byte(content)),
	}

	_, err := o.client.PutObject(&input)
	return err
}

func (o *ObjectStoreConfig) LogInfof(bucketName string, msgFormat string, a ...interface{}) {
	msg := fmt.Sprintf(msgFormat, a...)
	if o.enable {
		err := o.writeToObjectStore(bucketName, time.Now().Format(time.RFC3339Nano), "[INFO] "+msg)
		if err != nil {
			o.logger.Errorf("Error while writing to obj store: %w", err)
		}
	}
	o.logger.Info(msg)
}

func (o *ObjectStoreConfig) LogWarnf(bucketName string, msgFormat string, a ...interface{}) {
	msg := fmt.Sprintf(msgFormat, a...)
	if o.enable {
		err := o.writeToObjectStore(bucketName, time.Now().Format(time.RFC3339Nano), "[WARN] "+msg)
		if err != nil {
			o.logger.Errorf("Error while writing to obj store: %w", err)
		}
	}
	o.logger.Warn(msg)
}

func (o *ObjectStoreConfig) LogErrorf(bucketName string, msgFormat string, a ...interface{}) {
	msg := fmt.Sprintf(msgFormat, a...)
	if o.enable {
		err := o.writeToObjectStore(bucketName, time.Now().Format(time.RFC3339Nano), "[ERROR] "+msg)
		if err != nil {
			o.logger.Errorf("Error while writing to obj store: %w", err)
		}
	}
	o.logger.Error(msg)
}
