package webdav

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/symmecrypt/ciphers/aesgcm"
	"github.com/ovh/symmecrypt/convergent"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/webdav"
)

func TestWebdav(t *testing.T) {
	log.SetLogger(t)
	dir, err := ioutil.TempDir("", t.Name()+"-cdn-webdav-*")
	require.NoError(t, err)
	srv := &webdav.Handler{
		FileSystem: webdav.Dir(dir),
		LockSystem: webdav.NewMemLS(),
		Logger: func(r *http.Request, err error) {
			if err != nil {
				t.Logf("WEBDAV [%s]: %s, ERROR: %s\n", r.Method, r.URL, err)
			} else {
				t.Logf("WEBDAV [%s]: %s \n", r.Method, r.URL)
			}
		},
	}
	http.Handle("/", srv)
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", 8091), nil); err != nil {
			log.Fatalf("Error with WebDAV server: %v", err)
		}
	}()

	var ok bool
	for i := 0; i < 10; i++ {
		time.Sleep(time.Second)

		t.Logf("Checking if webdav server is started...\n")
		resp, err := http.Get("http://localhost:8091")
		if err != nil {
			t.Logf("webdav not started yet, err: %v\n", err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Logf("webdav not started yet code: %d\n", resp.StatusCode)
			continue
		}
		ok = true
		break
	}
	require.True(t, ok, "webdav server is not running")

	t.Logf("webdav server running\n")

	var driver = new(Webdav)
	err = driver.Init(context.TODO(), &storage.WebdavStorageConfiguration{
		Address:  "http://localhost:8091",
		Username: "username",
		Password: "password",
		Path:     "data",
		Encryption: []convergent.ConvergentEncryptionConfig{
			{
				Cipher:      aesgcm.CipherName,
				LocatorSalt: "secret_locator_salt",
				SecretValue: "secret_value",
			},
		},
	})
	require.NoError(t, err, "unable to initialiaze webdav driver")

	itemUnit := sdk.CDNItemUnit{
		Locator: "a_locator",
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

	btes, err := ioutil.ReadAll(r)
	require.NoError(t, err)
	err = r.Close()
	require.NoError(t, err)

	require.Equal(t, "something", string(btes))

	require.NoError(t, driver.Remove(context.TODO(), itemUnit))
}
