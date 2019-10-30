package objectstore_test

import (
	"context"
	"encoding/base64"
	"io"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/cdn/objectstore"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/stretchr/testify/require"
)

type TestStringObject struct {
	t *testing.T
	s string
	r io.Reader
}

func (t *TestStringObject) GetName() string {
	return base64.StdEncoding.EncodeToString([]byte(t.s))
}

func (t *TestStringObject) GetPath() string {
	return base64.StdEncoding.EncodeToString([]byte(t.t.Name()))
}

func (t *TestStringObject) Read(p []byte) (int, error) {
	if t.r == nil {
		t.r = strings.NewReader(t.s)
	}
	return t.r.Read(p)
}

func TestAwsS3_Store(t *testing.T) {
	log.SetLogger(t)
	cfg := test.LoadTestingConf(t)

	s3AccessKeyId := cfg["s3AccessKeyId"]
	s3BucketName := cfg["s3BucketName"]
	s3Prefix := cfg["s3Prefix"]
	s3Region := cfg["s3Region"]
	s3SecretAccessKey := cfg["s3SecretAccessKey"]
	s3SessionToken := cfg["s3SessionToken"]
	s3Endpoint := cfg["s3Endpoint"]
	s3DisableSSL := cfg["s3DisableSSL"]
	s3ForcePathStyle := cfg["s3ForcePathStyle"]

	if s3Endpoint == "" && s3AccessKeyId == "" {
		t.Logf("Unable to read aws s3 configuration. Skipping this tests.")
		t.SkipNow()
	}

	storeCfg := objectstore.Config{
		IntegrationName: "test",
		Kind:            objectstore.AWSS3,
	}

	storeCfg.Options.AWSS3 = objectstore.ConfigOptionsAWSS3{
		AccessKeyID:     s3AccessKeyId,
		BucketName:      s3BucketName,
		DisableSSL:      sdk.MustParseBool(s3DisableSSL),
		Endpoint:        s3Endpoint,
		ForcePathStyle:  sdk.MustParseBool(s3ForcePathStyle),
		Prefix:          s3Prefix,
		SecretAccessKey: s3SecretAccessKey,
		SessionToken:    s3SessionToken,
		Region:          s3Region,
	}

	store, err := objectstore.Init(context.TODO(), storeCfg)
	require.NoError(t, err)

	o := &TestStringObject{t: t, s: sdk.RandomString(10)}
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()

	_, err = store.Store(ctx, o, ioutil.NopCloser(o))
	require.NoError(t, err)

	//writer, err := store.Open(ctx, o)
	//require.NoError(t, err)
	//go func() {
	//	fmt.Println("starting write...")
	//	_, err = io.Copy(writer, o)
	//	fmt.Println("ending write...")
	//	require.NoError(t, err)
	//}()
	//err = writer.Close()
	//require.NoError(t, err)

}

func TestAwsS3_Writer(t *testing.T) {
	log.SetLogger(t)
	cfg := test.LoadTestingConf(t)

	s3AccessKeyId := cfg["s3AccessKeyId"]
	s3BucketName := cfg["s3BucketName"]
	s3Prefix := cfg["s3Prefix"]
	s3Region := cfg["s3Region"]
	s3SecretAccessKey := cfg["s3SecretAccessKey"]
	s3SessionToken := cfg["s3SessionToken"]
	s3Endpoint := cfg["s3Endpoint"]
	s3DisableSSL := cfg["s3DisableSSL"]
	s3ForcePathStyle := cfg["s3ForcePathStyle"]

	if s3Endpoint == "" && s3AccessKeyId == "" {
		t.Logf("Unable to read aws s3 configuration. Skipping this tests.")
		t.SkipNow()
	}

	storeCfg := objectstore.Config{
		IntegrationName: "test",
		Kind:            objectstore.AWSS3,
	}

	storeCfg.Options.AWSS3 = objectstore.ConfigOptionsAWSS3{
		AccessKeyID:     s3AccessKeyId,
		BucketName:      s3BucketName,
		DisableSSL:      sdk.MustParseBool(s3DisableSSL),
		Endpoint:        s3Endpoint,
		ForcePathStyle:  sdk.MustParseBool(s3ForcePathStyle),
		Prefix:          s3Prefix,
		SecretAccessKey: s3SecretAccessKey,
		SessionToken:    s3SessionToken,
		Region:          s3Region,
	}

	store, err := objectstore.Init(context.TODO(), storeCfg)
	require.NoError(t, err)

	o := &TestStringObject{t: t, s: sdk.RandomString(10)}
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()

	writer, err := store.Open(ctx, o)
	require.NoError(t, err)
	go func() {
		_, err := io.Copy(writer, o)
		require.NoError(t, err)
		err = writer.(*objectstore.S3File).EndWrite()
		require.NoError(t, err)
	}()
	err = writer.Close()
	require.NoError(t, err)

}
