package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/actionplugin"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

const dummyBinaryFile = "https://dl.plik.ovh/file/7d0jJMNKaFEFc8OI/A5GlNec32iismk9s/dummy_plugin"

func postFile(t *testing.T,
	db *gorp.DbMap,
	filename string,
	targetURL string,
	method string,
	handler Handler,
	check func(*testing.T, *gorp.DbMap, *httptest.ResponseRecorder)) {

	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	fileWriter, err := bodyWriter.CreateFormFile("UploadFile", filename)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	// open file handle
	fh, err := os.Open(filename)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	//iocopy
	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	req, err := http.NewRequest(method, targetURL, bodyBuf)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	c := &context.Ctx{
		User: &sdk.User{
			ID: 1,
		},
	}

	req.Header.Add("Content-Type", contentType)

	router := mux.NewRouter()
	router.HandleFunc(targetURL,
		func(w http.ResponseWriter, r *http.Request) {
			if err := handler(w, r, db, c); err != nil {
				WriteError(w, r, err)
			}
			t.Logf("Headers : %v", w.Header())
		})
	http.Handle(targetURL, router)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if check != nil {
		check(t, db, w)
	}
}

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

func TestAddPluginHandlerSuccess(t *testing.T) {
	db := test.SetupPG(t)

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
	objectstore.Initialize(cfg)

	u, _ := assets.InsertAdminUser(t, db)
	if err := actionplugin.Delete(db, "dummy", u.ID); err != nil {
		t.Log(err)
	}

	path, delete, err := downloadFile(t, "dummy", dummyBinaryFile)
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
		assert.Equal(t, "dummy", a.Name)
		assert.Equal(t, sdk.PluginAction, a.Type)
		assert.Equal(t, "This is a dummy plugin", a.Description)
		assert.Equal(t, "dummy", a.Requirements[0].Name)
		assert.Equal(t, sdk.PluginRequirement, a.Requirements[0].Type)
		assert.Equal(t, "dummy", a.Requirements[0].Value)
		assert.Empty(t, a.Actions)
		assert.True(t, a.Enabled)
		assert.Equal(t, "param1", a.Parameters[0].Name)
		assert.Equal(t, sdk.StringParameter, a.Parameters[0].Type)
		assert.Equal(t, "this is a parameter", a.Parameters[0].Description)
		assert.Equal(t, "value1", a.Parameters[0].Value)

	})

}

func TestAddPluginHandlerFailWithInvalidPlugin(t *testing.T) {
	db := test.SetupPG(t)

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
	objectstore.Initialize(cfg)

	u, _ := assets.InsertAdminUser(t, db)
	actionplugin.Delete(db, "dummy", u.ID)

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
	db := test.SetupPG(t)

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
	objectstore.Initialize(cfg)

	u, _ := assets.InsertAdminUser(t, db)
	actionplugin.Delete(db, "dummy", u.ID)

	path, delete, err := downloadFile(t, "dummy", dummyBinaryFile)
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
	db := test.SetupPG(t)

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
	objectstore.Initialize(cfg)

	u, _ := assets.InsertAdminUser(t, db)
	actionplugin.Delete(db, "dummy", u.ID)

	path, delete, err := downloadFile(t, "dummy", dummyBinaryFile)
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
	db := test.SetupPG(t)

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
	objectstore.Initialize(cfg)

	u, _ := assets.InsertAdminUser(t, db)
	actionplugin.Delete(db, "dummy", u.ID)

	path, delete, err := downloadFile(t, "dummy", dummyBinaryFile)
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

		c := &context.Ctx{
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
