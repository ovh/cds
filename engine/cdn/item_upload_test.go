package cdn

import (
	"context"
	"crypto/md5"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdn"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/symmecrypt/ciphers/aesgcm"
	"github.com/ovh/symmecrypt/convergent"
	"github.com/ovh/symmecrypt/keyloader"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	cdntest "github.com/ovh/cds/engine/cdn/test"
)

func TestPostUploadHandler(t *testing.T) {
	s, db := newTestService(t)
	s.Cfg.EnableLogProcessing = true

	cfg := test.LoadTestingConf(t, sdk.TypeCDN)

	cdntest.ClearItem(t, context.TODO(), s.Mapper, db)
	cdntest.ClearItem(t, context.TODO(), s.Mapper, db)

	// Start CDN
	ctx := context.Background()
	tmpDir, err := ioutil.TempDir("", t.Name()+"-cdn-1-*")
	require.NoError(t, err)

	tmpDir2, err := ioutil.TempDir("", t.Name()+"-cdn-2-*")
	require.NoError(t, err)

	t.Logf(tmpDir)
	t.Logf(tmpDir2)
	cdnUnits, err := storage.Init(ctx, s.Mapper, db.DbMap, sdk.NewGoRoutines(), storage.Configuration{
		SyncSeconds:     10,
		HashLocatorSalt: "thisismysalt",
		Buffers: []storage.BufferConfiguration{
			{
				Name: "refis_buffer",
				Redis: &storage.RedisBufferConfiguration{
					Host:     cfg["redisHost"],
					Password: cfg["redisPassword"],
				},
				BufferType: storage.CDNBufferTypeLog,
			},
			{
				Name: "local_buffer",
				Local: &storage.LocalBufferConfiguration{
					Path: tmpDir,
					Encryption: []*keyloader.KeyConfig{
						{
							Key:        "jesuisunecle",
							Cipher:     aesgcm.CipherName,
							Identifier: "local-bukker-id",
						},
					},
				},
				BufferType: storage.CDNBufferTypeFile,
			},
		},
		Storages: []storage.StorageConfiguration{
			{
				Name: "local_storage",
				Local: &storage.LocalStorageConfiguration{
					Path: tmpDir2,
					Encryption: []convergent.ConvergentEncryptionConfig{
						{
							Cipher:      aesgcm.CipherName,
							LocatorSalt: "secret_locator_salt",
							SecretValue: "secret_value",
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	s.Units = cdnUnits

	// Mock cds client
	s.Client = cdsclient.New(cdsclient.Config{Host: "http://lolcat.api", InsecureSkipVerifyTLS: false})
	gock.InterceptClient(s.Client.(cdsclient.Raw).HTTPClient())
	t.Cleanup(gock.OffAll)

	// Create worker
	workerKey, err := jws.NewRandomSymmetricKey(32)
	require.NoError(t, err)
	jobRunID := int64(1)
	worker := sdk.Worker{
		ID:         "1",
		Name:       "myworker",
		JobRunID:   &jobRunID,
		PrivateKey: []byte(base64.StdEncoding.EncodeToString(workerKey)),
	}

	// Mock get worker
	gock.New("http://lolcat.api").AddMatcher(func(r *http.Request, rr *gock.Request) (bool, error) {
		b, err := gock.MatchPath(r, rr)
		assert.NoError(t, err)
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.String(), "http://lolcat.api/worker/myworker?withKey=true") {
			if b {
				return true, nil
			}
			return false, nil
		}
		return false, nil
	}).Reply(http.StatusOK).JSON(worker)

	workerSignature := cdn.Signature{
		Timestamp:    time.Now().Unix(),
		ProjectKey:   "projKey",
		WorkflowID:   1,
		JobID:        1,
		JobName:      "my job",
		RunID:        1,
		WorkflowName: "my workflow",
		Worker: &cdn.SignatureWorker{
			WorkerID:     worker.ID,
			WorkerName:   worker.Name,
			ArtifactName: "myartifact",
		},
	}
	signer, err := jws.NewHMacSigner(workerKey)
	require.NoError(t, err)

	signature, err := jws.Sign(signer, workerSignature)
	require.NoError(t, err)

	// Create artifact
	fileContent := []byte("Hi, I am foo")
	myartifact, errF := os.Create(path.Join(os.TempDir(), "myartifact"))
	defer os.RemoveAll(path.Join(os.TempDir(), "myartifact"))
	require.NoError(t, errF)
	_, errW := myartifact.Write(fileContent)
	require.NoError(t, errW)

	errClose := myartifact.Close()
	require.NoError(t, errClose)

	f, err := os.Open(path.Join(os.TempDir(), "myartifact"))
	require.NoError(t, err)
	defer f.Close()
	h := md5.New()
	_, err = io.Copy(h, f)
	require.NoError(t, err)
	md5Sum := hex.EncodeToString(h.Sum(nil))
	require.NoError(t, f.Close())

	f, err = os.Open(path.Join(os.TempDir(), "myartifact"))
	require.NoError(t, err)
	defer f.Close()
	hasher := sha512.New()
	_, err = io.Copy(hasher, f)
	require.NoError(t, err)
	sha := hex.EncodeToString(hasher.Sum(nil))
	require.NoError(t, f.Close())

	vars := map[string]string{}
	uri := s.Router.GetRoute("POST", s.postUploadHandler, vars)
	require.NotEmpty(t, uri)

	moreHeaders := map[string]string{
		"X-CDS-WORKER-SIGNATURE": signature,
	}
	req := assets.NewMultipartRequest(t, "POST", uri, path.Join(os.TempDir(), "myartifact"), "file", "myartifact", nil, moreHeaders)
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)
	for _, r := range gock.Pending() {
		t.Logf("PENDING: %s \n", r.Request().URLStruct.String())
	}

	require.Equal(t, 204, rec.Code)
	require.True(t, gock.IsDone())

	its, err := item.LoadAll(ctx, s.Mapper, db, 1, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)

	require.Equal(t, 1, len(its))
	require.Equal(t, sha, its[0].Hash)
	require.Equal(t, md5Sum, its[0].MD5)
	require.Equal(t, int64(len(fileContent)), its[0].Size)
	require.Equal(t, sdk.CDNStatusItemCompleted, its[0].Status)
}
