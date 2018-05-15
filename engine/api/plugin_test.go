package api

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/ovh/cds/engine/api/test"
)

const dummyBinaryFile = "https://github.com/ovh/cds/releases/download/0.8.1/plugin-download-" + runtime.GOOS + "-amd64"

func downloadFile(t *testing.T, name, url string) (string, func(), error) {
	t.Logf("Downloading file %s", url)

	resp, err := http.Get(url)
	test.NoError(t, err)
	if err != nil {
		t.Skipf("Unable to download file %s", err)
		return "", nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	test.NoError(t, err)
	if err != nil {
		t.Fail()
		return "", nil, err
	}

	path := path.Join(os.TempDir(), name)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0744)
	test.NoError(t, err)
	if err != nil {
		t.Fail()
		return "", nil, err
	}

	t.Logf("Writing file  to %s", path)
	_, err = io.Copy(f, bytes.NewBuffer(data))
	test.NoError(t, err)
	if err != nil {
		t.Fail()
		return "", nil, err
	}

	return path, func() {
		t.Logf("Delete file %s\n", path)
		os.RemoveAll(path)
	}, nil
}

/*
func TestAddPluginHandlerSuccess(t *testing.T) {
	api, db, router := newTestAPI(t)

	basedir, err := ioutil.TempDir("", "cds-test")
	if err != nil {
		t.Fail()
		return
	}
	defer func() {
		os.RemoveAll(basedir)
	}()

	cfg := objectstore.Config{
		Kind: objectstore.Filesystem,
		Options: objectstore.ConfigOptions{
			Filesystem: objectstore.ConfigOptionsFilesystem{
				Basedir: basedir,
			},
		},
	}
	c := ctx.Background()
	objectstore.Initialize(c, cfg)

	u, _ := assets.InsertAdminUser(api.mustDB())
	if err := actionplugin.Delete(api.mustDB(), "plugin-download", u.ID); err != nil {
		t.Log(err)
	}

	path, delete, err := downloadFile(t, "plugin-download", dummyBinaryFile)
	if delete != nil {
		defer delete()
	}
	if err != nil {
		t.Fail()
		return
	}

	postFile(t, db, path, "/plugin_test/TestAddPluginHandlerSuccess", "POST", addPluginHandler, func(t *testing.T, db *gorp.DbMap, resp *httptest.ResponseRecorder) {
		t.Logf("Code status : %d", resp.Code)
		assert.Equal(t, 201, resp.Code)
		body, _ := ioutil.ReadAll(resp.Body)
		t.Logf(string(body))
		a := &sdk.Action{}
		err := json.Unmarshal(body, a)
		if err != nil {
			t.Fail()
			return
		}
		assert.Equal(t, "plugin-download", a.Name)
		assert.Equal(t, sdk.PluginAction, a.Type)
		assert.Equal(t, "This is a plugin to download file from URL", a.Description)
		assert.Equal(t, "plugin-download", a.Requirements[0].Name)
		assert.Equal(t, sdk.PluginRequirement, a.Requirements[0].Type)
		assert.Equal(t, "plugin-download", a.Requirements[0].Value)
		assert.Empty(t, a.Actions)
		assert.True(t, a.Enabled)

		var checked bool
		for _, v := range a.Parameters {
			if v.Name == "filepath" {
				assert.Equal(t, sdk.StringParameter, v.Type)
				assert.Equal(t, ".", v.Value)
				assert.Equal(t, "the destination of your file to be copied", v.Description)
				checked = true
			}
		}

		assert.True(t, checked, "no parameter checked on plugin-download")
	})

}

func TestAddPluginHandlerFailWithInvalidPlugin(t *testing.T) {
	api, db, router := newTestAPI(t)

	basedir, err := ioutil.TempDir("", "cds-test")
	if err != nil {
		t.Fail()
		return
	}
	defer func() {
		os.RemoveAll(basedir)
	}()

	cfg := objectstore.Config{
		Kind: objectstore.Filesystem,
		Options: objectstore.ConfigOptions{
			Filesystem: objectstore.ConfigOptionsFilesystem{
				Basedir: basedir,
			},
		},
	}
	c := ctx.Background()
	objectstore.Initialize(c, cfg)

	u, _ := assets.InsertAdminUser(api.mustDB())
	actionplugin.Delete(api.mustDB(), "plugin-download", u.ID)

	path, delete, err := downloadFile(t, "dummy1", dummyBinaryFile)
	if delete != nil {
		defer delete()
	}
	if err != nil {
		t.Fail()
		return
	}

	postFile(t, db, path, "/plugin_test/TestAddPluginHandlerFailWithInvalidPlugin", "POST", addPluginHandler, func(t *testing.T, db *gorp.DbMap, resp *httptest.ResponseRecorder) {
		body, _ := ioutil.ReadAll(resp.Body)
		t.Logf("Code status : %d", resp.Code)
		t.Logf("Response : %s", string(body))
		assert.Equal(t, 400, resp.Code)
	})
}

func TestAddPluginHandlerFailWithConflict(t *testing.T) {
	api, db, router := newTestAPI(t)

	basedir, err := ioutil.TempDir("", "cds-test")
	if err != nil {
		t.Fail()
		return
	}
	defer func() {
		os.RemoveAll(basedir)
	}()

	cfg := objectstore.Config{
		Kind: objectstore.Filesystem,
		Options: objectstore.ConfigOptions{
			Filesystem: objectstore.ConfigOptionsFilesystem{
				Basedir: basedir,
			},
		},
	}
	c := ctx.Background()
	objectstore.Initialize(c, cfg)

	u, _ := assets.InsertAdminUser(api.mustDB())
	actionplugin.Delete(api.mustDB(), "plugin-download", u.ID)

	path, delete, err := downloadFile(t, "plugin-download", dummyBinaryFile)
	if delete != nil {
		defer delete()
	}
	if err != nil {
		t.Fail()
		return
	}

	postFile(t, db, path, "/plugin_test/TestAddPluginHandlerFailWithConflict", "POST", addPluginHandler, func(t *testing.T, db *gorp.DbMap, resp *httptest.ResponseRecorder) {
		body, _ := ioutil.ReadAll(resp.Body)
		t.Logf("Code status : %d", resp.Code)
		t.Logf("Response : %s", string(body))
		assert.Equal(t, 201, resp.Code)

		postFile(t, db, path, "/plugin_test/TestAddPluginHandlerFailWithConflictBis", "POST", addPluginHandler, func(t *testing.T, db *gorp.DbMap, resp *httptest.ResponseRecorder) {
			body, _ := ioutil.ReadAll(resp.Body)
			t.Logf("Code status : %d", resp.Code)
			t.Logf("Response : %s", string(body))
			assert.Equal(t, 409, resp.Code)
		})
	})
}

func TestUpdatePluginHandlerSuccess(t *testing.T) {
	api, db, router := newTestAPI(t)

	basedir, err := ioutil.TempDir("", "cds-test")
	if err != nil {
		t.Fail()
		return
	}

	defer func() {
		os.RemoveAll(basedir)
	}()

	cfg := objectstore.Config{
		Kind: objectstore.Filesystem,
		Options: objectstore.ConfigOptions{
			Filesystem: objectstore.ConfigOptionsFilesystem{
				Basedir: basedir,
			},
		},
	}
	c := ctx.Background()
	objectstore.Initialize(c, cfg)

	u, _ := assets.InsertAdminUser(api.mustDB())
	actionplugin.Delete(api.mustDB(), "plugin-download", u.ID)

	path, delete, err := downloadFile(t, "plugin-download", dummyBinaryFile)
	if delete != nil {
		defer delete()
	}
	if err != nil {
		t.Fail()
		return
	}

	//First create the action
	postFile(t, db, path, "/plugin_test/TestUpdatePluginHandlerSuccess_POST", "POST", addPluginHandler, func(t *testing.T, db *gorp.DbMap, resp *httptest.ResponseRecorder) {
		t.Logf("Code status : %d", resp.Code)
		assert.Equal(t, 201, resp.Code)
		body, _ := ioutil.ReadAll(resp.Body)
		t.Logf(string(body))
		a := &sdk.Action{}
		err := json.Unmarshal(body, a)
		if err != nil {
			t.Fail()
			return
		}
		//Then update the action
		postFile(t, db, path, "/plugin_test/TestUpdatePluginHandlerSuccess_PUT", "PUT", updatePluginHandler, func(t *testing.T, db *gorp.DbMap, resp *httptest.ResponseRecorder) {
			t.Logf("Code status : %d", resp.Code)
			assert.Equal(t, 200, resp.Code)
			body, _ := ioutil.ReadAll(resp.Body)
			t.Logf(string(body))
			a := &sdk.Action{}
			err := json.Unmarshal(body, a)
			if err != nil {
				t.Fail()
				return
			}
		})
	})
}

func TestDeletePluginHandlerSuccess(t *testing.T) {
	api, db, router := newTestAPI(t)

	basedir, err := ioutil.TempDir("", "cds-test")
	if err != nil {
		t.Fail()
		return
	}

	defer func() {
		os.RemoveAll(basedir)
	}()

	cfg := objectstore.Config{
		Kind: objectstore.Filesystem,
		Options: objectstore.ConfigOptions{
			Filesystem: objectstore.ConfigOptionsFilesystem{
				Basedir: basedir,
			},
		},
	}
	c := ctx.Background()
	objectstore.Initialize(c, cfg)

	u, _ := assets.InsertAdminUser(api.mustDB())
	actionplugin.Delete(api.mustDB(), "plugin-download", u.ID)

	path, delete, err := downloadFile(t, "plugin-download", dummyBinaryFile)
	if delete != nil {
		defer delete()
	}
	if err != nil {
		t.Fail()
		return
	}

	//First create the action
	postFile(t, db, path, "/plugin_test/TestDeletePluginHandlerSuccess_POST", "POST", addPluginHandler, func(t *testing.T, db *gorp.DbMap, resp *httptest.ResponseRecorder) {
		t.Logf("Code status : %d", resp.Code)
		assert.Equal(t, 201, resp.Code)
		body, _ := ioutil.ReadAll(resp.Body)
		t.Logf(string(body))
		a := &sdk.Action{}
		err := json.Unmarshal(body, a)
		if err != nil {
			t.Fail()
			return
		}

		targetURL := "/plugin_test/TestDeletePluginHandlerSuccess/{name}"
		req, err := http.NewRequest("DELETE", strings.Replace(targetURL, "{name}", a.Name, -1), nil)
		if err != nil {
			t.Error(err)
			t.Fail()
			return
		}

		c := &businesscontext.Ctx{
			User: &sdk.User{
				ID: 1,
			},
		}

		router := mux.NewRouter()
		router.HandleFunc(targetURL,
			func(w http.ResponseWriter, r *http.Request) {
				deletePluginHandler(w, r, db, c)
				t.Logf("Headers : %v", w.Header())
			})
		http.Handle(targetURL, router)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})
}
*/
