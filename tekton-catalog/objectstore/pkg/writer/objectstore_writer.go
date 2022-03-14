/*
Copyright [2022] [IBM]

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

package writer

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"sync"

	"github.com/IBM/ibm-cos-sdk-go/aws"
	"github.com/IBM/ibm-cos-sdk-go/aws/credentials"
	"github.com/IBM/ibm-cos-sdk-go/aws/session"
	"github.com/IBM/ibm-cos-sdk-go/service/s3"
)

type ObjectStoreWriterConfig struct {
	CreateBucket      bool
	DefaultBucketName string
	AccessKey         string
	SecretKey         string
	Region            string
	ServiceEndpoint   string
	Token             string
	S3ForcePathStyle  bool
}

type Writer struct {
	DefaultBucketName string
	buffer            *bytes.Buffer
	client            *s3.S3
	mu                sync.Mutex
}

func (w *Writer) load(o ObjectStoreWriterConfig) error {
	cosCredentials := credentials.NewStaticCredentials(o.AccessKey, o.SecretKey, o.Token)
	// Create client config
	var conf = aws.NewConfig().
		WithRegion(o.Region).
		WithEndpoint(o.ServiceEndpoint).
		WithCredentials(cosCredentials).
		WithS3ForcePathStyle(o.S3ForcePathStyle)

	var sess = session.Must(session.NewSession())
	w.client = s3.New(sess, conf)
	input := &s3.CreateBucketInput{
		Bucket: aws.String(o.DefaultBucketName),
	}
	w.DefaultBucketName = o.DefaultBucketName
	if o.CreateBucket {
		_, err := w.client.CreateBucket(input)
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) CreateNewBucket(bucketName string) error {
	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}
	_, err := w.client.CreateBucket(input)
	return err
}

func compress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write(data); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (w *Writer) write(pipelineRunName, pipelineTaskName, resultName string, content []byte) error {
	bucketName := fmt.Sprintf("%s/%s/%s/", w.DefaultBucketName, pipelineRunName, pipelineTaskName)
	err := w.CreateNewBucket(bucketName)
	if err != nil {
		// TODO check if the failure is due to bucket already exists.
	}
	compressed, err := compress(content)
	key := fmt.Sprintf("%s.tgz", resultName)
	input := s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
		Body:   bytes.NewReader(compressed),
	}

	_, err = w.client.PutObject(&input)
	return err
}

func (w *Writer) writeToObjectStore(bucketName string, key string, content []byte) error {
	input := s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
		Body:   bytes.NewReader(content),
	}

	_, err := w.client.PutObject(&input)
	return err
}
