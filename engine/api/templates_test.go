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
	"os"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/testwithdb"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/objectstore"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"

	"github.com/ovh/cds/sdk"
)

const (
	testTestTemplate = "https://dl.plik.ovh/file/blHntG6HDsMt2C3E/rBatDljsct2eDpL3/testtemplate"
)

func Test_getTemplatesHandler(t *testing.T) {
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := testwithdb.SetupPG(t)
	assert.NoError(t, err)

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local", TTL: 30})
	router = &Router{authDriver, mux.NewRouter(), "/Test_getTemplatesHandler"}
	if router.mux == nil {
		t.Fatal("Router cannot be nil")
		return
	}
	router.init()

	//Create admin user
	u, pass, err := testwithdb.InsertAdminUser(t, db)
	assert.NoError(t, err)

	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	//Prepare request
	vars := map[string]string{}
	uri := router.getRoute("GET", getTemplatesHandler, vars)
	if uri == "" {
		t.Fail()
		return
	}
	req, err := http.NewRequest("GET", uri, nil)
	testwithdb.AuthentifyRequest(t, req, u, pass)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())
}

func Test_addTemplateHandler(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := testwithdb.SetupPG(t)
	assert.NoError(t, err)

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local", TTL: 30})
	router = &Router{authDriver, mux.NewRouter(), "/Test_addTemplateHandler"}
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
	objectstore.Initialize("filesystem", "", "", "", tmpDir)

	defer os.RemoveAll(tmpDir)

	//Create admin user
	u, pass, err := testwithdb.InsertAdminUser(t, db)
	assert.NoError(t, err)

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
	if uri == "" {
		t.Fail()
		return
	}
	req, err := http.NewRequest("POST", uri, bodyBuf)
	testwithdb.AuthentifyRequest(t, req, u, pass)

	req.Header.Add("Content-Type", contentType)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	t.Logf("Body: %s", w.Body.String())

	templ := sdk.TemplateExtention{}
	json.Unmarshal(w.Body.Bytes(), &templ)

	//Prepare request
	vars = map[string]string{}
	uri = router.getRoute("GET", getTemplatesHandler, vars)
	if uri == "" {
		t.Fail()
		return
	}
	req, err = http.NewRequest("GET", uri, nil)
	testwithdb.AuthentifyRequest(t, req, u, pass)

	//Do the request
	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	templs := []sdk.TemplateExtention{}
	json.Unmarshal(w.Body.Bytes(), &templs)

	assert.EqualValues(t, []sdk.TemplateExtention{templ}, templs)

	assert.Equal(t, 200, w.Code)

	dbmap := database.DBMap(db)
	dbmap.Delete(&templ)
}

func Test_deleteTemplateHandler(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := testwithdb.SetupPG(t)
	assert.NoError(t, err)

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local", TTL: 30})
	router = &Router{authDriver, mux.NewRouter(), "/Test_deleteTemplateHandler"}
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
	objectstore.Initialize("filesystem", "", "", "", tmpDir)

	defer os.RemoveAll(tmpDir)

	//Create admin user
	u, pass, err := testwithdb.InsertAdminUser(t, db)
	assert.NoError(t, err)

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
	if uri == "" {
		t.Fail()
		return
	}
	req, err := http.NewRequest("POST", uri, bodyBuf)
	testwithdb.AuthentifyRequest(t, req, u, pass)

	req.Header.Add("Content-Type", contentType)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	t.Logf("Body: %s", w.Body.String())

	templ := sdk.TemplateExtention{}
	json.Unmarshal(w.Body.Bytes(), &templ)

	//Prepare request
	vars = map[string]string{
		"id": fmt.Sprintf("%d", templ.ID),
	}
	uri = router.getRoute("DELETE", deleteTemplateHandler, vars)
	if uri == "" {
		t.Fail()
		return
	}
	req, err = http.NewRequest("DELETE", uri, nil)
	testwithdb.AuthentifyRequest(t, req, u, pass)

	//Do the request
	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	t.Logf("Body: %s", w.Body.String())
}

func Test_updateTemplateHandler(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := testwithdb.SetupPG(t)
	assert.NoError(t, err)

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local", TTL: 30})
	router = &Router{authDriver, mux.NewRouter(), "/Test_addUpdateHandler"}
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
	objectstore.Initialize("filesystem", "", "", "", tmpDir)

	defer os.RemoveAll(tmpDir)

	//Create admin user
	u, pass, err := testwithdb.InsertAdminUser(t, db)
	assert.NoError(t, err)

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
	if uri == "" {
		t.Fail()
		return
	}
	req, err := http.NewRequest("POST", uri, bodyBuf)
	testwithdb.AuthentifyRequest(t, req, u, pass)

	req.Header.Add("Content-Type", contentType)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	t.Logf("Body: %s", w.Body.String())

	templ := sdk.TemplateExtention{}
	json.Unmarshal(w.Body.Bytes(), &templ)

	//Prepare request
	vars = map[string]string{}
	uri = router.getRoute("GET", getTemplatesHandler, vars)
	if uri == "" {
		t.Fail()
		return
	}
	req, err = http.NewRequest("GET", uri, nil)
	testwithdb.AuthentifyRequest(t, req, u, pass)

	//Do the request
	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	templs := []sdk.TemplateExtention{}
	json.Unmarshal(w.Body.Bytes(), &templs)

	assert.EqualValues(t, []sdk.TemplateExtention{templ}, templs)

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
	if uri == "" {
		t.Fail()
		return
	}
	req, err = http.NewRequest("PUT", uri, bodyBuf)
	testwithdb.AuthentifyRequest(t, req, u, pass)

	req.Header.Add("Content-Type", contentType)

	//Do the request
	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	t.Logf("Body: %s", w.Body.String())

	templ = sdk.TemplateExtention{}
	json.Unmarshal(w.Body.Bytes(), &templ)

	dbmap := database.DBMap(db)
	dbmap.Delete(&templ)
}

func Test_getBuildTemplatesHandler(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := testwithdb.SetupPG(t)
	assert.NoError(t, err)

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local", TTL: 30})
	router = &Router{authDriver, mux.NewRouter(), "/Test_getBuildTemplatesHandler"}
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
	objectstore.Initialize("filesystem", "", "", "", tmpDir)

	defer os.RemoveAll(tmpDir)

	//Create admin user
	u, pass, err := testwithdb.InsertAdminUser(t, db)
	assert.NoError(t, err)

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
	if uri == "" {
		t.Fail()
		return
	}
	req, err := http.NewRequest("POST", uri, bodyBuf)
	testwithdb.AuthentifyRequest(t, req, u, pass)

	req.Header.Add("Content-Type", contentType)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	t.Logf("Body: %s", w.Body.String())

	templ := sdk.TemplateExtention{}
	json.Unmarshal(w.Body.Bytes(), &templ)

	//Prepare request
	vars = map[string]string{}
	uri = router.getRoute("GET", getBuildTemplatesHandler, vars)
	if uri == "" {
		t.Fail()
		return
	}
	req, err = http.NewRequest("GET", uri, nil)
	testwithdb.AuthentifyRequest(t, req, u, pass)

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

	dbmap := database.DBMap(db)
	dbmap.Delete(&templ)

}

func Test_applyTemplatesHandler(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := testwithdb.SetupPG(t)
	assert.NoError(t, err)

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local", TTL: 30})
	router = &Router{authDriver, mux.NewRouter(), "/Test_getBuildTemplatesHandler"}
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
	objectstore.Initialize("filesystem", "", "", "", tmpDir)

	defer os.RemoveAll(tmpDir)

	//Create admin user
	u, pass, err := testwithdb.InsertAdminUser(t, db)
	assert.NoError(t, err)

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
	if uri == "" {
		t.Fail()
		return
	}
	req, err := http.NewRequest("POST", uri, bodyBuf)
	testwithdb.AuthentifyRequest(t, req, u, pass)

	req.Header.Add("Content-Type", contentType)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	templ := sdk.TemplateExtention{}
	json.Unmarshal(w.Body.Bytes(), &templ)

	//Insert a new project
	pKey := testwithdb.RandomString(t, 10)
	testwithdb.InsertTestProject(t, db, pKey, pKey)

	//Prepare the data send on applyTemplatesHandler
	opts := sdk.ApplyTemplatesOptions{
		ApplicationName: testwithdb.RandomString(t, 10),
		ApplicationVariables: map[string]string{
			"repo": "git@github.com:ovh/cds.git",
		},
		BuildTemplateName: templ.Name,
		BuildTemplateParams: []sdk.TemplateParam{
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
	uri = router.getRoute("POST", applyTemplatesHandler, vars)
	if uri == "" {
		t.Fail()
		return
	}
	req, err = http.NewRequest("POST", uri, bodyBuf)
	testwithdb.AuthentifyRequest(t, req, u, pass)

	req.Header.Add("Content-Type", contentType)

	//Do the request
	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("body: %s", w.Body.String())

	dbmap := database.DBMap(db)
	dbmap.Delete(&templ)
}
