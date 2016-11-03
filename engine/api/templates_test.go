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
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/testwithdb"
	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/objectstore"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
)

const testTestTemplate = "https://dl.plik.ovh/file/DEiIopA4denn54ts/95PHIfkfGpJb52Dl/testtemplate"

func Test_getTemplatesHandler(t *testing.T) {
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := testwithdb.SetupPG(t)
	assert.NoError(t, err)

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local"})
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

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local"})
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

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local"})
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
		"id": templ.ID,
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

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local"})
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
		"id": templ.ID,
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
