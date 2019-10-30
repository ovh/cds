package objectstore

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// AWSS3Store implements ObjectStore interface with filesystem driver
type AWSS3Store struct {
	projectIntegration sdk.ProjectIntegration
	prefix             string
	bucketName         string
	sess               *session.Session
}

func newS3Store(integration sdk.ProjectIntegration, conf ConfigOptionsAWSS3) (*AWSS3Store, error) {
	log.Info("ObjectStore> Initialize AWS S3 driver for bucket: %s in region %s", conf.BucketName, conf.Region)
	aConf := aws.NewConfig()
	aConf.Region = aws.String(conf.Region)
	if conf.AuthFromEnvironment {
		aConf.Credentials = credentials.NewEnvCredentials()
	} else if conf.Profile != "" {
		// if the shared creds file is empty the AWS SDK will check the defaults automatically
		aConf.Credentials = credentials.NewSharedCredentials(conf.SharedCredsFile, conf.Profile)
	} else {
		aConf.Credentials = credentials.NewStaticCredentials(conf.AccessKeyID, conf.SecretAccessKey, conf.SessionToken)
	}
	// If a custom endpoint is set, set up a new endPoint resolver (eg. minio)
	if conf.Endpoint != "" {
		aConf.Endpoint = aws.String(conf.Endpoint)
		aConf.DisableSSL = aws.Bool(conf.DisableSSL)
		aConf.S3ForcePathStyle = aws.Bool(conf.ForcePathStyle)
	}
	sess, err := session.NewSession(aConf)
	if err != nil {
		return nil, sdk.WrapError(err, "Unable to create an AWS session")
	}
	return &AWSS3Store{
		bucketName:         conf.BucketName,
		prefix:             conf.Prefix,
		projectIntegration: integration,
		sess:               sess,
	}, nil
}

func (s *AWSS3Store) account() (*s3.ListObjectsOutput, error) {
	log.Debug("AWS-S3-Store> Getting bucket info")
	s3n := s3.New(s.sess)
	out, err := s3n.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(s.bucketName),
		Prefix: aws.String(s.prefix),
	})
	if err != nil {
		return nil, sdk.WrapError(err, "AWS-S3-Store> Unable to read data from input object")
	}

	return out, nil
}

func (s *AWSS3Store) getObjectPath(o Object) string {
	return path.Join(s.prefix, o.GetPath(), o.GetName())
}

func (s *AWSS3Store) TemporaryURLSupported() bool {
	return true
}

func (s *AWSS3Store) GetProjectIntegration() sdk.ProjectIntegration {
	return s.projectIntegration
}

func (s *AWSS3Store) Status() sdk.MonitoringStatusLine {
	out, err := s.account()
	if err != nil {
		return sdk.MonitoringStatusLine{Component: "Object-Store", Value: "AWSS3 KO" + err.Error(), Status: sdk.MonitoringStatusAlert}
	}
	return sdk.MonitoringStatusLine{
		Component: "Object-Store",
		Value:     fmt.Sprintf("S3 OK (%d objects)", len(out.Contents)),
		Status:    sdk.MonitoringStatusOK,
	}
}

func (s *AWSS3Store) Open(ctx context.Context, o Object) (io.WriteCloser, error) {
	reader, writer := io.Pipe()
	channel := make(chan []byte, 1000)
	s3file := S3File{
		ctx:     ctx,
		s3Store: s,
		channel: channel,
		uploadInput: s3manager.UploadInput{
			Key:    aws.String(s.getObjectPath(o)),
			Bucket: aws.String(s.bucketName),
			Body:   reader,
		},
		writer: writer,
	}

	go func() {
		for btes := range s3file.channel {
			if _, err := writer.Write(btes); err != nil {
				log.Error("cannot write in writer %v", err)
			}
		}
		if err := writer.Close(); err != nil {
			log.Error("cannot close writer : %v", err)
		}
	}()

	return &s3file, nil
}

func (s *AWSS3Store) Store(ctx context.Context, o Object, data io.ReadCloser) (string, error) {
	defer data.Close()
	log.Debug("AWS-S3-Store> Setting up uploader")
	uploader := s3manager.NewUploader(s.sess)
	b, e := ioutil.ReadAll(data)
	if e != nil {
		return "", sdk.WrapError(e, "AWS-S3-Store> Unable to read data from input object")
	}

	log.Debug("AWS-S3-Store> Uploading object %s to bucket %s", s.getObjectPath(o), s.bucketName)

	out, err := uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Key:    aws.String(s.getObjectPath(o)),
		Bucket: aws.String(s.bucketName),
		Body:   bytes.NewReader(b),
	})
	if err != nil {
		return "", sdk.WrapError(err, "AWS-S3-Store> Unable to create object %s", s.getObjectPath(o))
	}
	log.Debug("AWS-S3-Store> Successfully uploaded object %s to bucket %s", s.getObjectPath(o), s.bucketName)
	return out.Location, nil
}

// StoreURL returns a temporary url and a secret key to store an object
func (s *AWSS3Store) StoreURL(o Object, contentType string) (string, string, error) {
	log.Debug("AWS-S3-Store> StoreURL")
	s3n := s3.New(s.sess)
	key := aws.String(s.getObjectPath(o))
	req, _ := s3n.PutObjectRequest(&s3.PutObjectInput{
		Key:    key,
		Bucket: aws.String(s.bucketName),
	})

	if contentType != "" {
		req.HTTPRequest.Header.Set("Content-Type", contentType)
	}

	urlStr, _, err := req.PresignRequest(5 * time.Minute)
	if err != nil {
		return "", "", sdk.WrapError(err, "failed to sign request")
	}
	log.Debug("AWS-S3-Store> StoreURL urlStr:%v", urlStr)
	return urlStr, *key, nil
}

func (s *AWSS3Store) Fetch(ctx context.Context, o Object) (io.ReadCloser, error) {
	s3n := s3.New(s.sess)
	log.Debug("AWS-S3-Store> Fetching object %s from bucket %s", s.getObjectPath(o), s.bucketName)
	out, err := s3n.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Key:    aws.String(s.getObjectPath(o)),
		Bucket: aws.String(s.bucketName),
	})
	if err != nil {
		return nil, sdk.WrapError(err, "AWS-S3-Store> Unable to download object %s", s.getObjectPath(o))
	}
	log.Debug("AWS-S3-Store> Successfully retrieved object %s from bucket %s", s.getObjectPath(o), s.bucketName)
	return out.Body, nil
}

func (s *AWSS3Store) Delete(ctx context.Context, o Object) error {
	s3n := s3.New(s.sess)
	log.Debug("AWS-S3-Store> Deleting object %s from bucket %s", s.getObjectPath(o), s.bucketName)
	_, err := s3n.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Key:    aws.String(s.getObjectPath(o)),
		Bucket: aws.String(s.bucketName),
	})
	if err != nil {
		return sdk.WrapError(err, "AWS-S3-Store> Unable to delete object %s", s.getObjectPath(o))
	}
	log.Debug("AWS-S3-Store> Successfully Deleted object %s/%s", s.bucketName, s.getObjectPath(o))
	return nil
}

// FetchURL returns a temporary url and a secret key to fetch an object
func (s *AWSS3Store) FetchURL(o Object) (string, string, error) {
	log.Debug("AWS-S3-Store> FetchURL")
	s3n := s3.New(s.sess)
	key := aws.String(s.getObjectPath(o))
	req, _ := s3n.GetObjectRequest(&s3.GetObjectInput{
		Key:    key,
		Bucket: aws.String(s.bucketName),
	})
	urlStr, err := req.Presign(5 * time.Minute)
	if err != nil {
		return "", "", sdk.WrapError(err, "failed to sign request")
	}
	log.Debug("AWS-S3-Store> FetchURL urlStr:%v key:%v", urlStr, *key)
	return urlStr, *key, nil
}

// ServeStaticFiles is not implemented on s3
func (s *AWSS3Store) ServeStaticFiles(o Object, entrypoint string, data io.ReadCloser) (string, error) {
	return "", sdk.WithStack(sdk.ErrNotImplemented)
}

// ServeStaticFilesURL returns a temporary url and a secret key to serve static files in a container
func (s *AWSS3Store) ServeStaticFilesURL(o Object, entrypoint string) (string, string, error) {
	return "", "", sdk.WithStack(sdk.ErrNotImplemented)
}

// S3File represent a file in S3 with writer interface implementation
type S3File struct {
	ctx         context.Context
	s3Store     *AWSS3Store
	uploadInput s3manager.UploadInput
	writer      *io.PipeWriter
	reader      *io.PipeReader
	channel     chan []byte
}

func (s3file *S3File) Write(p []byte) (int, error) {
	s3file.channel <- p
	return len(p), nil
}

func (s3file *S3File) Close() error {
	close(s3file.channel)
	uploader := s3manager.NewUploader(s3file.s3Store.sess)
	_, err := uploader.UploadWithContext(s3file.ctx, &s3file.uploadInput)
	return sdk.WrapError(err, "AWS-S3-Store> Unable to create object %s", *s3file.uploadInput.Key)
}
