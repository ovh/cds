package s3

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/symmecrypt/ciphers/aesgcm"
	"github.com/ovh/symmecrypt/convergent"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
)

// To run the test, run the make minio_start && make minio_reset_bucket from the tests directory
// Then export the mentioned env variables: S3_BUCKET, AWS_DEFAULT_REGION, MINIO_ACCESS_KEY, MINIO_SECRET_KEY and AWS_ENDPOINT_URL
// If not set, the test is skipped
func TestS3(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	var driver = new(S3)
	err := driver.Init(context.TODO(), &storage.S3StorageConfiguration{
		BucketName:      os.Getenv("S3_BUCKET"),
		Region:          os.Getenv("AWS_DEFAULT_REGION"),
		Prefix:          "tests",
		AccessKeyID:     os.Getenv("MINIO_ACCESS_KEY"),
		SecretAccessKey: os.Getenv("MINIO_SECRET_KEY"),
		Endpoint:        os.Getenv("AWS_ENDPOINT_URL"),
		DisableSSL:      true,
		ForcePathStyle:  true,
		Encryption: []convergent.ConvergentEncryptionConfig{
			{
				Cipher:      aesgcm.CipherName,
				LocatorSalt: "secret_locator_salt",
				SecretValue: "secret_value",
			},
		},
	})
	if err != nil && strings.Contains(err.Error(), "MissingRegion") {
		t.Logf("skipping this test: %v", err)
		t.SkipNow()
	}
	require.NoError(t, err, "unable to initialize s3 driver")

	itemUnit := sdk.CDNItemUnit{
		Locator: "a_locator",
		Item: &sdk.CDNItem{
			Type: sdk.CDNTypeItemStepLog,
		},
	}
	w, err := driver.NewWriter(context.TODO(), itemUnit)
	require.NoError(t, err)
	require.NotNil(t, w)

	_, err = w.Write([]byte("something"))
	require.NoError(t, err)

	err = w.Close()
	require.NoError(t, err)

	r, err := driver.NewReader(context.TODO(), itemUnit)
	require.NoError(t, err)
	require.NotNil(t, r)

	btes, err := io.ReadAll(r)
	require.NoError(t, err)
	err = r.Close()
	require.NoError(t, err)

	require.Equal(t, "something", string(btes))
}
