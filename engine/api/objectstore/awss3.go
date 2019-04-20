package objectstore

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// FilesystemStore implements ObjectStore interface with filesystem driver
type AWSS3Store struct {
	projectIntegration sdk.ProjectIntegration
	prefix             string
	bucketName         string
	region             string
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
	log.Debug("AWS-S3-Store> Getting bucket info\n")
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
	return false
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

func (s *AWSS3Store) Store(o Object, data io.ReadCloser) (string, error) {
	defer data.Close()
	log.Debug("AWS-S3-Store> Setting up uploader\n")
	uploader := s3manager.NewUploader(s.sess)
	b, e := ioutil.ReadAll(data)
	if e != nil {
		return "", sdk.WrapError(e, "AWS-S3-Store> Unable to read data from input object")
	}

	log.Debug("AWS-S3-Store> Uploading object %s to bucket %s\n", s.getObjectPath(o), s.bucketName)
	out, err := uploader.Upload(&s3manager.UploadInput{
		Key:    aws.String(s.getObjectPath(o)),
		Bucket: aws.String(s.bucketName),
		Body:   bytes.NewReader(b),
	})
	if err != nil {
		return "", sdk.WrapError(err, "AWS-S3-Store> Unable to create object %s", s.getObjectPath(o))
	}
	log.Debug("AWS-S3-Store> Successfully uploaded object %s to bucket %s\n", s.getObjectPath(o), s.bucketName)
	return out.Location, nil
}

func (s *AWSS3Store) ServeStaticFiles(o Object, entrypoint string, data io.ReadCloser) (string, error) {
	return "", sdk.ErrNotImplemented
}

func (s *AWSS3Store) Fetch(o Object) (io.ReadCloser, error) {
	s3n := s3.New(s.sess)
	log.Debug("AWS-S3-Store> Fetching object %s from bucket %s\n", s.getObjectPath(o), s.bucketName)
	out, err := s3n.GetObject(&s3.GetObjectInput{
		Key:    aws.String(s.getObjectPath(o)),
		Bucket: aws.String(s.bucketName),
	})
	if err != nil {
		return nil, sdk.WrapError(err, "AWS-S3-Store> Unable to download object %s", s.getObjectPath(o))
	}
	log.Debug("AWS-S3-Store> Successfully retrieved object %s from bucket %s\n", s.getObjectPath(o), s.bucketName)
	return out.Body, nil
}

func (s *AWSS3Store) Delete(o Object) error {
	s3n := s3.New(s.sess)
	log.Debug("AWS-S3-Store> Deleting object %s from bucket %s\n", s.getObjectPath(o), s.bucketName)
	_, err := s3n.DeleteObject(&s3.DeleteObjectInput{
		Key:    aws.String(s.getObjectPath(o)),
		Bucket: aws.String(s.bucketName),
	})
	if err != nil {
		return sdk.WrapError(err, "AWS-S3-Store> Unable to delete object %s", s.getObjectPath(o))
	}
	log.Debug("AWS-S3-Store> Successfully Deleted object %s/%s\n", s.bucketName, s.getObjectPath(o))
	return nil
}
