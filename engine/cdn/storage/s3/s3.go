package s3

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	s3session "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/cdn/storage/encryption"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

type S3 struct {
	client *s3session.Session
	storage.AbstractUnit
	encryption.ConvergentEncryption
	config storage.S3StorageConfiguration
}

var (
	_ storage.StorageUnit = new(S3)
)

const driverName = "s3"

func init() {
	storage.RegisterDriver(driverName, new(S3))
}

func (s *S3) GetDriverName() string {
	return driverName
}

func (s *S3) Init(_ context.Context, cfg interface{}) error {
	config, is := cfg.(*storage.S3StorageConfiguration)
	if !is {
		return sdk.WithStack(fmt.Errorf("invalid configuration: %T", cfg))
	}
	s.config = *config
	s.ConvergentEncryption = encryption.New(config.Encryption)

	aConf := aws.NewConfig()
	aConf.Region = aws.String(config.Region)
	if config.AuthFromEnvironment {
		aConf.Credentials = credentials.NewEnvCredentials()
	} else if config.Profile != "" { // if the shared creds file is empty the AWS SDK will check the defaults automatically
		aConf.Credentials = credentials.NewSharedCredentials(config.SharedCredsFile, config.Profile)
	} else {
		aConf.Credentials = credentials.NewStaticCredentials(config.AccessKeyID, config.SecretAccessKey, config.SessionToken)
	}

	// If a custom endpoint is set, set up a new endPoint resolver (eg. minio)
	if config.Endpoint != "" {
		aConf.Endpoint = aws.String(config.Endpoint)
		aConf.DisableSSL = aws.Bool(config.DisableSSL)
		aConf.S3ForcePathStyle = aws.Bool(config.ForcePathStyle)
	}

	sess, err := session.NewSession(aConf)
	if err != nil {
		return sdk.WrapError(err, "Unable to create an AWS session")
	}

	s.client = sess
	c := s3.New(s.client)
	_, err = c.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(s.config.BucketName),
		Prefix: aws.String(s.config.Prefix),
	})

	return sdk.WithStack(err)
}

func (s *S3) ItemExists(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, i sdk.CDNItem) (bool, error) {
	iu, err := s.ExistsInDatabase(ctx, m, db, i.ID)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return false, nil
		}
		return false, err
	}

	objectName := s.getObjectName(*iu)

	c := s3.New(s.client)

	req, resp := c.ListObjectsV2Request(&s3.ListObjectsV2Input{Bucket: &s.config.BucketName, Prefix: &objectName})
	if err := req.Send(); err != nil {
		return false, sdk.WrapError(err, "unable to send s3 request: %v", err)
	}

	return len(resp.Contents) >= 0, nil
}

type s3WriteCloser struct {
	pw          *io.PipeWriter
	uploadError error
}

func (s *s3WriteCloser) Write(btes []byte) (int, error) {
	return s.pw.Write(btes)
}

func (s *s3WriteCloser) Close() error {
	if s.uploadError != nil {
		return s.pw.CloseWithError(s.uploadError)
	}
	return s.pw.Close()
}

var _ io.WriteCloser = new(s3WriteCloser)

func (s *S3) NewWriter(ctx context.Context, i sdk.CDNItemUnit) (io.WriteCloser, error) {
	object := s.getObjectName(i)

	log.Debug(ctx, "[%T] writing to %s", s, object)

	uploader := s3manager.NewUploader(s.client)
	uploader.Concurrency = 1
	uploader.PartSize = s3manager.MinUploadPartSize

	pr, pw := io.Pipe()
	w := &s3WriteCloser{pw: pw}
	go func() {
		_, err := uploader.Upload(&s3manager.UploadInput{
			Bucket: &s.config.BucketName,
			Key:    &object,
			Body:   pr,
		})
		if err != nil {
			w.uploadError = err
		}
	}()

	return w, nil
}

func (s *S3) NewReader(ctx context.Context, i sdk.CDNItemUnit) (io.ReadCloser, error) {
	object := s.getObjectName(i)
	log.Debug(ctx, "[%T] reading from %s", s, object)

	c := s3.New(s.client)
	output, err := c.GetObject(&s3.GetObjectInput{
		Bucket: &s.config.BucketName,
		Key:    &object,
	})

	if err != nil {
		return nil, sdk.WithStack(err)
	}

	return output.Body, nil
}

func (s *S3) getObjectName(i sdk.CDNItemUnit) string {
	loc := i.Locator
	path := fmt.Sprintf("%s-%s-%s", s.config.Prefix, i.Item.Type, loc)
	return escape(path)
}

func escape(s string) string {
	s = url.QueryEscape(s)
	s = strings.Replace(s, "/", "-", -1)
	return s
}

// Status returns the status of swift account
func (s *S3) Status(ctx context.Context) []sdk.MonitoringStatusLine {
	c := s3.New(s.client)
	out, err := c.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(s.config.BucketName),
		Prefix: aws.String(s.config.Prefix),
	})
	if err != nil {
		return []sdk.MonitoringStatusLine{{Component: "backend/" + s.Name(), Value: "S3 KO" + err.Error(), Status: sdk.MonitoringStatusAlert}}
	}
	return []sdk.MonitoringStatusLine{{
		Component: "backend/" + s.Name(),
		Value:     fmt.Sprintf("S3 OK (%d objects)", len(out.Contents)),
		Status:    sdk.MonitoringStatusOK,
	}}
}

func (s *S3) Remove(ctx context.Context, i sdk.CDNItemUnit) error {
	object := s.getObjectName(i)
	c := s3.New(s.client)
	_, err := c.DeleteObject(&s3.DeleteObjectInput{
		Bucket: &s.config.BucketName,
		Key:    &object,
	})
	return sdk.WithStack(err)
}
