package internal

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generatePrivateKey creates a RSA Private Key of specified byte size
func generatePrivateKey(t *testing.T, bitSize int) *rsa.PrivateKey {
	// Private Key generation
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	require.NoError(t, err)

	// Validate Private Key
	err = privateKey.Validate()
	require.NoError(t, err)

	return privateKey
}

// encodePrivateKeyToPEM encodes Private Key from RSA to PEM format
func encodePrivateKeyToPEM(privateKey *rsa.PrivateKey) []byte {
	// Get ASN.1 DER format
	privDER := x509.MarshalPKCS1PrivateKey(privateKey)

	// pem.Block
	privBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privDER,
	}

	// Private key in PEM format
	privatePEM := pem.EncodeToMemory(&privBlock)

	return privatePEM
}

func TestInstallKey_SSHKeyWithoutDestination(t *testing.T) {
	var w = new(CurrentWorker)
	fs := afero.NewOsFs()
	basedir := "test-" + test.GetTestName(t) + "-" + sdk.RandomString(10) + "-" + fmt.Sprintf("%d", time.Now().Unix())
	require.NoError(t, fs.MkdirAll(basedir, os.FileMode(0755)))

	if err := w.Init("test-worker", "test-hatchery", "http://lolcat.host", "xxx-my-token", "", true, afero.NewBasePathFs(fs, basedir)); err != nil {
		t.Fatalf("worker init failed: %v", err)
	}

	require.NoError(t, w.BaseDir().Mkdir("keys", os.FileMode(0700)))
	_, err := w.BaseDir().Stat("keys")
	require.NoError(t, err)

	keyDir, err := w.BaseDir().Open("keys")
	require.NoError(t, err)
	keyDir.Close()

	w.currentJob.context = workerruntime.SetKeysDirectory(context.TODO(), keyDir)

	priKey := generatePrivateKey(t, 2048)
	priKeyPEM := encodePrivateKeyToPEM(priKey)
	priKeyVar := sdk.Variable{
		Name:  "my-ssh-key",
		Type:  string(sdk.KeyTypeSSH),
		Value: string(priKeyPEM),
	}

	resp, err := w.InstallKey(priKeyVar)
	require.NoError(t, err)

	content, err := ioutil.ReadFile(resp.PKey)
	require.NoError(t, err)
	assert.Equal(t, string(priKeyPEM), string(content))

	kPath := filepath.Join(keyDir.Name(), "my-ssh-key")
	t.Logf("expecting the key to be written at: %s", kPath)
	_, err = w.BaseDir().Stat(kPath)
	require.NoError(t, err)
}

func TestInstallKey_SSHKeyWithRelativeDestination(t *testing.T) {
	var w = new(CurrentWorker)
	fs := afero.NewOsFs()
	basedir := "test-" + test.GetTestName(t) + "-" + sdk.RandomString(10) + "-" + fmt.Sprintf("%d", time.Now().Unix())
	require.NoError(t, fs.MkdirAll(basedir, os.FileMode(0755)))

	if err := w.Init("test-worker", "test-hatchery", "http://lolcat.host", "xxx-my-token", "", true, afero.NewBasePathFs(fs, basedir)); err != nil {
		t.Fatalf("worker init failed: %v", err)
	}

	require.NoError(t, w.BaseDir().Mkdir("keys", os.FileMode(0700)))
	keyDir, err := w.BaseDir().Open("keys")
	require.NoError(t, err)

	w.currentJob.context = workerruntime.SetKeysDirectory(context.TODO(), keyDir)

	priKey := generatePrivateKey(t, 2048)
	priKeyPEM := encodePrivateKeyToPEM(priKey)
	priKeyVar := sdk.Variable{
		Name:  "my-ssh-key",
		Type:  string(sdk.KeyTypeSSH),
		Value: string(priKeyPEM),
	}

	resp, err := w.InstallKeyTo(priKeyVar, "ssh/id_rsa")
	require.NoError(t, err)

	content, err := ioutil.ReadFile(resp.PKey)
	require.NoError(t, err)
	assert.Equal(t, string(priKeyPEM), string(content))
	t.Logf("the path to the key is %s", resp.PKey)

	keyDirPath, _ := w.BaseDir().(*afero.BasePathFs).RealPath(".")
	absKeyDirPath, _ := filepath.Abs(keyDirPath)
	assert.Equal(t, filepath.Join(absKeyDirPath, "ssh/id_rsa"), resp.PKey)

	kPath := "ssh/id_rsa"
	t.Logf("expecting the key to be written at: %s", kPath)
	_, err = w.BaseDir().Stat(kPath)
	require.NoError(t, err)
}

func TestInstallKey_SSHKeyWithAbsoluteDestination(t *testing.T) {
	var w = new(CurrentWorker)

	if err := w.Init("test-worker", "test-hatchery", "http://lolcat.host", "xxx-my-token", "", true, nil); err != nil {
		t.Fatalf("worker init failed: %v", err)
	}

	priKey := generatePrivateKey(t, 2048)
	priKeyPEM := encodePrivateKeyToPEM(priKey)
	priKeyVar := sdk.Variable{
		Name:  "my-ssh-key",
		Type:  string(sdk.KeyTypeSSH),
		Value: string(priKeyPEM),
	}

	resp, err := w.InstallKeyTo(priKeyVar, "/tmp/fake_id_rsa")
	require.NoError(t, err)

	content, err := ioutil.ReadFile(resp.PKey)
	require.NoError(t, err)
	assert.Equal(t, string(priKeyPEM), string(content))
	assert.Equal(t, "/tmp/fake_id_rsa", resp.PKey)

	t.Logf("expecting the key to be written at /tmp/fake_id_rsa")
	_, err = os.Stat("/tmp/fake_id_rsa")
	require.NoError(t, err)
	os.RemoveAll("/tmp/fake_id_rsa")
}
