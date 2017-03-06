package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/templateextension"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

const (
	testTestTemplate = "https://dl.plik.ovh/file/FIcfha7CCqHO8DON/c69ILIhdO4iq73GH/testtemplate"
)

func Test_getTemplatesHandler(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)

	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/Test_getTemplatesHandler"}
	if router.mux == nil {
		t.Fatal("Router cannot be nil")
		return
	}
	router.init()

	//Create admin user
	u, pass := assets.InsertAdminUser(t, db)

	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	//Prepare request
	vars := map[string]string{}
	uri := router.getRoute("GET", getTemplatesHandler, vars)
	test.NotEmpty(t, uri)

	req, _ := http.NewRequest("GET", uri, nil)
	assets.AuthentifyRequest(t, req, u, pass)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())
}

func Test_addTemplateHandler(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)

	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/Test_addTemplateHandler"}
	if router.mux == nil {
		t.Fatal("Router cannot be nil")
		return
	}
	router.init()

	tmpDir, err := ioutil.TempDir("objectstore", "test")
	if err != nil {
		t.Fatal(err)
		return
	}
	cfg := objectstore.Config{
		Kind: objectstore.Filesystem,
		Options: objectstore.ConfigOptions{
			Filesystem: objectstore.ConfigOptionsFilesystem{
				Basedir: tmpDir,
			},
		},
	}
	objectstore.Initialize(cfg)

	defer os.RemoveAll(tmpDir)

	//Create admin user
	u, pass := assets.InsertAdminUser(t, db)
	test.NoError(t, err)

	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	downloadPublicAction(t, u, pass)

	//download the binary from plik
	path, delete, err := downloadFile(t, "testtemplate", testTestTemplate)
	if delete != nil {
		defer delete()
	}
	if err != nil {
		t.Fail()
		return
	}

	//prepare upload
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	fileWriter, err := bodyWriter.CreateFormFile("UploadFile", path)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	// open file handle
	fh, err := os.Open(path)
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

	//Prepare request
	vars := map[string]string{}
	uri := router.getRoute("POST", addTemplateHandler, vars)
	test.NotEmpty(t, uri)

	req, err := http.NewRequest("POST", uri, bodyBuf)
	assets.AuthentifyRequest(t, req, u, pass)

	req.Header.Add("Content-Type", contentType)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	t.Logf("Body: %s", w.Body.String())

	templ := sdk.TemplateExtension{}
	json.Unmarshal(w.Body.Bytes(), &templ)

	//Prepare request
	vars = map[string]string{}
	uri = router.getRoute("GET", getTemplatesHandler, vars)
	test.NotEmpty(t, uri)

	req, err = http.NewRequest("GET", uri, nil)
	assets.AuthentifyRequest(t, req, u, pass)

	//Do the request
	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	templs := []sdk.TemplateExtension{}
	json.Unmarshal(w.Body.Bytes(), &templs)

	dbtempl := templateextension.TemplateExtension(templ)

	if _, err := db.Delete(&dbtempl); err != nil {
		t.Fatal(err)
	}

	assert.EqualValues(t, []sdk.TemplateExtension{templ}, templs)

	assert.Equal(t, 200, w.Code)

}

func Test_deleteTemplateHandler(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)

	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/Test_deleteTemplateHandler"}
	if router.mux == nil {
		t.Fatal("Router cannot be nil")
		return
	}
	router.init()

	tmpDir, err := ioutil.TempDir("objectstore", "test")
	if err != nil {
		t.Fatal(err)
		return
	}
	cfg := objectstore.Config{
		Kind: objectstore.Filesystem,
		Options: objectstore.ConfigOptions{
			Filesystem: objectstore.ConfigOptionsFilesystem{
				Basedir: tmpDir,
			},
		},
	}
	objectstore.Initialize(cfg)

	defer os.RemoveAll(tmpDir)

	//Create admin user
	u, pass := assets.InsertAdminUser(t, db)
	test.NoError(t, err)

	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	downloadPublicAction(t, u, pass)

	//download the binary from plik
	path, delete, err := downloadFile(t, "testtemplate", testTestTemplate)
	if delete != nil {
		defer delete()
	}
	if err != nil {
		t.Fail()
		return
	}

	//prepare upload
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	fileWriter, err := bodyWriter.CreateFormFile("UploadFile", path)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	// open file handle
	fh, err := os.Open(path)
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

	//Prepare request
	vars := map[string]string{}
	uri := router.getRoute("POST", addTemplateHandler, vars)
	test.NotEmpty(t, uri)

	req, err := http.NewRequest("POST", uri, bodyBuf)
	assets.AuthentifyRequest(t, req, u, pass)

	req.Header.Add("Content-Type", contentType)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	t.Logf("Body: %s", w.Body.String())

	templ := sdk.TemplateExtension{}
	json.Unmarshal(w.Body.Bytes(), &templ)

	//Prepare request
	vars = map[string]string{
		"id": fmt.Sprintf("%d", templ.ID),
	}
	uri = router.getRoute("DELETE", deleteTemplateHandler, vars)
	test.NotEmpty(t, uri)

	req, err = http.NewRequest("DELETE", uri, nil)
	assets.AuthentifyRequest(t, req, u, pass)

	//Do the request
	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	t.Logf("Body: %s", w.Body.String())
}

func Test_updateTemplateHandler(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)

	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/Test_addUpdateHandler"}
	router.init()

	tmpDir, err := ioutil.TempDir("objectstore", "test")
	if err != nil {
		t.Fatal(err)
		return
	}
	cfg := objectstore.Config{
		Kind: objectstore.Filesystem,
		Options: objectstore.ConfigOptions{
			Filesystem: objectstore.ConfigOptionsFilesystem{
				Basedir: tmpDir,
			},
		},
	}
	objectstore.Initialize(cfg)
	defer os.RemoveAll(tmpDir)

	//Create admin user
	u, pass := assets.InsertAdminUser(t, db)
	test.NoError(t, err)

	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	//download the binary from plik
	path, delete, err := downloadFile(t, "testtemplate", testTestTemplate)
	if delete != nil {
		defer delete()
	}
	if err != nil {
		t.Fail()
		return
	}

	//prepare upload
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	fileWriter, err := bodyWriter.CreateFormFile("UploadFile", path)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	// open file handle
	fh, err := os.Open(path)
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

	//Prepare request
	vars := map[string]string{}
	uri := router.getRoute("POST", addTemplateHandler, vars)
	test.NotEmpty(t, uri)

	req, err := http.NewRequest("POST", uri, bodyBuf)
	assets.AuthentifyRequest(t, req, u, pass)

	req.Header.Add("Content-Type", contentType)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	t.Logf("Body: %s", w.Body.String())

	templ := sdk.TemplateExtension{}
	json.Unmarshal(w.Body.Bytes(), &templ)

	//Prepare request
	vars = map[string]string{}
	uri = router.getRoute("GET", getTemplatesHandler, vars)
	test.NotEmpty(t, uri)

	req, err = http.NewRequest("GET", uri, nil)
	assets.AuthentifyRequest(t, req, u, pass)

	//Do the request
	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	templs := []sdk.TemplateExtension{}
	json.Unmarshal(w.Body.Bytes(), &templs)

	assert.EqualValues(t, []sdk.TemplateExtension{templ}, templs)

	assert.Equal(t, 200, w.Code)

	//Do the update
	//prepare upload
	bodyBuf = &bytes.Buffer{}
	bodyWriter = multipart.NewWriter(bodyBuf)

	fileWriter, err = bodyWriter.CreateFormFile("UploadFile", path)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	// open file handle
	fh, err = os.Open(path)
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

	contentType = bodyWriter.FormDataContentType()
	bodyWriter.Close()

	//Prepare request
	vars = map[string]string{
		"id": fmt.Sprintf("%d", templ.ID),
	}
	uri = router.getRoute("PUT", updateTemplateHandler, vars)
	test.NotEmpty(t, uri)

	req, err = http.NewRequest("PUT", uri, bodyBuf)
	assets.AuthentifyRequest(t, req, u, pass)

	req.Header.Add("Content-Type", contentType)

	//Do the request
	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	t.Logf("Body: %s", w.Body.String())

	templ = sdk.TemplateExtension{}
	json.Unmarshal(w.Body.Bytes(), &templ)

	dbtempl := templateextension.TemplateExtension(templ)
	db.Delete(&dbtempl)
}

func Test_getBuildTemplatesHandler(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)

	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/Test_getBuildTemplatesHandler"}
	router.init()

	tmpDir, err := ioutil.TempDir("objectstore", "test")
	if err != nil {
		t.Fatal(err)
		return
	}
	cfg := objectstore.Config{
		Kind: objectstore.Filesystem,
		Options: objectstore.ConfigOptions{
			Filesystem: objectstore.ConfigOptionsFilesystem{
				Basedir: tmpDir,
			},
		},
	}
	objectstore.Initialize(cfg)

	defer os.RemoveAll(tmpDir)

	//Create admin user
	u, pass := assets.InsertAdminUser(t, db)
	test.NoError(t, err)

	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	//download the binary from plik
	path, delete, err := downloadFile(t, "testtemplate", testTestTemplate)
	if delete != nil {
		defer delete()
	}
	if err != nil {
		t.Fail()
		return
	}

	//prepare upload
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	fileWriter, err := bodyWriter.CreateFormFile("UploadFile", path)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	// open file handle
	fh, err := os.Open(path)
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

	//Prepare request
	vars := map[string]string{}
	uri := router.getRoute("POST", addTemplateHandler, vars)
	test.NotEmpty(t, uri)

	req, err := http.NewRequest("POST", uri, bodyBuf)
	assets.AuthentifyRequest(t, req, u, pass)

	req.Header.Add("Content-Type", contentType)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	t.Logf("Body: %s", w.Body.String())

	templ := sdk.TemplateExtension{}
	json.Unmarshal(w.Body.Bytes(), &templ)

	//Prepare request
	vars = map[string]string{}
	uri = router.getRoute("GET", getBuildTemplatesHandler, vars)
	test.NotEmpty(t, uri)

	req, err = http.NewRequest("GET", uri, nil)
	assets.AuthentifyRequest(t, req, u, pass)

	//Do the request
	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	t.Logf("Body: %s", w.Body.String())

	templs := []sdk.Template{}
	json.Unmarshal(w.Body.Bytes(), &templs)

	assert.Equal(t, 2, len(templs))
	assert.Equal(t, "Void", templs[0].Name)
	assert.Equal(t, "testtemplate", templs[1].Name)
	assert.NotEmpty(t, templs[1].Params)

	dbtempl := templateextension.TemplateExtension(templ)
	db.Delete(&dbtempl)

}

func Test_applyTemplatesHandler(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)

	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/Test_applyTemplatesHandler"}
	if router.mux == nil {
		t.Fatal("Router cannot be nil")
		return
	}
	router.init()

	tmpDir, err := ioutil.TempDir("objectstore", "test")
	if err != nil {
		t.Fatal(err)
		return
	}
	cfg := objectstore.Config{
		Kind: objectstore.Filesystem,
		Options: objectstore.ConfigOptions{
			Filesystem: objectstore.ConfigOptionsFilesystem{
				Basedir: tmpDir,
			},
		},
	}
	objectstore.Initialize(cfg)

	defer os.RemoveAll(tmpDir)

	/*
	* CREATE AN ADMIN USER
	 */

	//Create admin user
	u, pass := assets.InsertAdminUser(t, db)
	test.NoError(t, err)

	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	downloadPublicAction(t, u, pass)

	//download the binary from plik
	path, delete, err := downloadFile(t, "testtemplate", testTestTemplate)
	if delete != nil {
		defer delete()
	}
	if err != nil {
		t.Fail()
		return
	}

	//prepare upload
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	fileWriter, err := bodyWriter.CreateFormFile("UploadFile", path)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	// open file handle
	fh, err := os.Open(path)
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

	//Prepare request
	vars := map[string]string{}
	uri := router.getRoute("POST", addTemplateHandler, vars)
	test.NotEmpty(t, uri)

	req, err := http.NewRequest("POST", uri, bodyBuf)
	assets.AuthentifyRequest(t, req, u, pass)

	req.Header.Add("Content-Type", contentType)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	templ := sdk.TemplateExtension{}
	json.Unmarshal(w.Body.Bytes(), &templ)

	/*
	* CREATE THE PROJECT
	 */

	//Insert a new project
	pKey := assets.RandomString(t, 10)
	p := assets.InsertTestProject(t, db, pKey, pKey)
	//Insert a Production environment
	environment.InsertEnvironment(db, &sdk.Environment{
		ProjectKey: pKey,
		ProjectID:  p.ID,
		Name:       "Production",
	})

	/*
	* APPLY THE TEMPLATE
	 */

	//Prepare the data send on applyTemplatesHandler
	opts := sdk.ApplyTemplatesOptions{
		ApplicationName: assets.RandomString(t, 10),
		TemplateName:    templ.Name,
		TemplateParams: []sdk.TemplateParam{
			{
				Name:  templ.Params[0].Name,
				Value: "value1",
			},
			{
				Name:  templ.Params[1].Name,
				Value: "value2",
			},
		},
	}

	btes, _ := json.Marshal(opts)
	bodyBuf = bytes.NewBuffer(btes)

	//Prepare request
	vars = map[string]string{
		"permProjectKey": pKey,
	}
	uri = router.getRoute("POST", applyTemplateHandler, vars)
	test.NotEmpty(t, uri)

	req, err = http.NewRequest("POST", uri, bodyBuf)
	assets.AuthentifyRequest(t, req, u, pass)

	req.Header.Add("Content-Type", contentType)

	//Do the request
	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("body: %s", w.Body.String())

	/*
	* APPLY THE TEMPLATE ON THE APPLICATION (second handler)
	 */

	//Prepare request
	vars = map[string]string{
		"key": pKey,
		"permApplicationName": opts.ApplicationName,
	}
	uri = router.getRoute("POST", applyTemplateOnApplicationHandler, vars)
	test.NotEmpty(t, uri)

	bodyBuf = bytes.NewBuffer(btes)
	req, err = http.NewRequest("POST", uri, bodyBuf)
	assets.AuthentifyRequest(t, req, u, pass)

	req.Header.Add("Content-Type", contentType)

	//Do the request
	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("body: %s", w.Body.String())

	dbtempl := templateextension.TemplateExtension(templ)
	db.Delete(&dbtempl)
}

func downloadPublicAction(t *testing.T, u *sdk.User, pass string) {
	//Load the gitclone public action
	//Prepare request
	uri := router.getRoute("POST", importActionHandler, nil)
	test.NotEmpty(t, uri)

	req, _ := http.NewRequest("POST", uri, nil)
	req.Form = url.Values{}
	req.Form.Add("url", "https://raw.githubusercontent.com/ovh/cds/master/contrib/actions/cds-git-clone.hcl")
	assets.AuthentifyRequest(t, req, u, pass)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)
	assert.True(t, w.Code >= 200)
}
