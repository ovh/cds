package internal

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"

	"github.com/magiconair/properties/assert"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/spf13/afero"
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

func TestInstallKey_SSHKeyWithoutDesination(t *testing.T) {
	var w = new(CurrentWorker)
	fs := afero.NewOsFs()
	basedir := "test-" + sdk.RandomString(10)
	require.NoError(t, fs.MkdirAll(basedir, os.FileMode(0755)))

	if err := w.Init("test-worker", "test-hatchery", "http://lolcat.host", "xxx-my-token", "", true, afero.NewBasePathFs(fs, basedir)); err != nil {
		t.Fatalf("worker init failed: %v", err)
	}

	require.NoError(t, w.Workspace().Mkdir("keys", os.FileMode(0700)))
	keyDir, err := w.Workspace().Open("keys")
	require.NoError(t, err)

	w.currentJob.context = workerruntime.SetKeysDirectory(context.TODO(), keyDir)

	priKey := generatePrivateKey(t, 2048)
	priKeyPEM := encodePrivateKeyToPEM(priKey)
	priKeyVar := sdk.Variable{
		Name:  "my-ssh-key",
		Type:  sdk.KeyTypeSSH,
		Value: string(priKeyPEM),
	}

	resp, err := w.InstallKey(priKeyVar, "")
	require.NoError(t, err)

	content, err := afero.ReadFile(w.Workspace(), resp.PKey)
	require.NoError(t, err)
	assert.Equal(t, string(priKeyPEM), string(content))
}

func TestInstallKey_SSHKeyWithDesination(t *testing.T) {
	var w = new(CurrentWorker)
	fs := afero.NewOsFs()
	basedir := "test-" + sdk.RandomString(10)
	require.NoError(t, fs.MkdirAll(basedir, os.FileMode(0755)))

	if err := w.Init("test-worker", "test-hatchery", "http://lolcat.host", "xxx-my-token", "", true, afero.NewBasePathFs(fs, basedir)); err != nil {
		t.Fatalf("worker init failed: %v", err)
	}

	require.NoError(t, w.Workspace().Mkdir("keys", os.FileMode(0700)))
	keyDir, err := w.Workspace().Open("keys")
	require.NoError(t, err)

	w.currentJob.context = workerruntime.SetKeysDirectory(context.TODO(), keyDir)

	priKey := generatePrivateKey(t, 2048)
	priKeyPEM := encodePrivateKeyToPEM(priKey)
	priKeyVar := sdk.Variable{
		Name:  "my-ssh-key",
		Type:  sdk.KeyTypeSSH,
		Value: string(priKeyPEM),
	}

	resp, err := w.InstallKey(priKeyVar, "id_rsa")
	require.NoError(t, err)

	content, err := afero.ReadFile(w.Workspace(), resp.PKey)
	require.NoError(t, err)
	assert.Equal(t, string(priKeyPEM), string(content))
}
