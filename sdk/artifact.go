package sdk

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Artifact define a file needed to be save for future use
type Artifact struct {
	ID          int64  `json:"id" cli:"id"`
	Project     string `json:"project"`
	Pipeline    string `json:"pipeline"`
	Application string `json:"application"`
	Environment string `json:"environment"`
	BuildNumber int    `json:"build_number"`
	Name        string `json:"name" cli:"name"`
	Tag         string `json:"tag"`

	DownloadHash     string `json:"download_hash" cli:"download_hash"`
	Size             int64  `json:"size,omitempty" cli:"size"`
	Perm             uint32 `json:"perm,omitempty"`
	MD5sum           string `json:"md5sum,omitempty" cli:"md5sum"`
	ObjectPath       string `json:"object_path,omitempty"`
	TempURL          string `json:"temp_url,omitempty"`
	TempURLSecretKey string `json:"-"`
}

// ArtifactsStore represents
type ArtifactsStore struct {
	Name                  string `json:"name"`
	Private               bool   `json:"private"`
	TemporaryURLSupported bool   `json:"temporary_url_supported"`
}

//GetName returns the name the artifact
func (a *Artifact) GetName() string {
	return a.Name
}

//GetPath returns the path of the artifact
func (a *Artifact) GetPath() string {
	container := fmt.Sprintf("%s-%s-%s-%s-%s", a.Project, a.Application, a.Environment, a.Pipeline, a.Tag)
	container = url.QueryEscape(container)
	container = strings.Replace(container, "/", "-", -1)
	return container
}

// Builtin artifact manipulation actions
const (
	ArtifactUpload   = "Artifact Upload"
	ArtifactDownload = "Artifact Download"
)

// Header name for artifact upload
const (
	ArtifactFileName = "ARTIFACT-FILENAME"
)

// DownloadArtifacts retrieves and download artifacts related to given project-pipeline-tag
// and download them into destdir
func DownloadArtifacts(project string, application string, pipeline string, tag string, destdir string, env string) error {
	arts, err := ListArtifacts(project, application, pipeline, tag, env)
	if err != nil {
		return err
	}

	for _, a := range arts {
		err := download(project, application, pipeline, a, destdir)
		if err != nil {
			return err
		}
	}

	return nil
}

func download(project, app, pip string, a Artifact, destdir string) error {
	var lasterr error
	var reader io.ReadCloser
	var doRequest func() (io.ReadCloser, int, error)

	if a.TempURL != "" {
		if verbose {
			fmt.Printf(">>> downloading artifact %s from %s\n", a.Name, a.TempURL)
		}
		doRequest = func() (io.ReadCloser, int, error) {
			req, err := http.NewRequest("GET", a.TempURL, nil)
			if err != nil {
				return nil, 0, err
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return nil, 0, err
			}

			return resp.Body, resp.StatusCode, nil
		}
	} else {
		doRequest = func() (io.ReadCloser, int, error) {
			uri := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/artifact/download/%d", project, app, pip, a.ID)
			return Stream("GET", uri, nil)
		}
	}

	for retry := 5; retry >= 0; retry-- {
		var code int
		var err error

		reader, code, err = doRequest()
		if err != nil {
			lasterr = err
			continue
		}
		defer reader.Close()

		//If internal server error... don't retry
		if code == 500 {
			lasterr = fmt.Errorf("HTTP %d", code)
			break
		}

		if code >= 300 {
			lasterr = fmt.Errorf("HTTP %d", code)
			continue
		}
	}

	if lasterr != nil {
		return lasterr
	}

	if err := os.MkdirAll(destdir, os.FileMode(0744)); err != nil {
		return err
	}

	destPath := path.Join(destdir, a.Name)

	mode := os.FileMode(0644)
	if a.Perm != uint32(0) {
		mode = os.FileMode(a.Perm)
	}

	f, erropen := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY, mode)
	if erropen != nil {
		return erropen
	}

	if _, err := io.Copy(f, reader); err != nil {
		return err
	}

	return f.Close()
}

// DownloadArtifact downloads a single artifact from API
func DownloadArtifact(project, app, pip, tag, destdir, env, filename string) error {
	tag = url.QueryEscape(tag)
	tag = strings.Replace(tag, "/", "-", -1)

	arts, err := ListArtifacts(project, app, pip, tag, env)
	if err != nil {
		return err
	}

	for _, a := range arts {
		if a.Name == filename {
			return download(project, app, pip, a, destdir)
		}
	}

	return fmt.Errorf("artifact not found")
}

// ListArtifacts retrieves the list of file stored as artifacts for given project-pipeline-tag
func ListArtifacts(project string, application string, pipeline string, tag string, env string) ([]Artifact, error) {
	tag = strings.Replace(tag, "/", "-", -1)
	tag = url.QueryEscape(tag)

	uri := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/artifact/%s?envName=%s", project, application, pipeline, tag, url.QueryEscape(env))
	data, code, err := Request("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	if code == http.StatusNotFound {
		return nil, fmt.Errorf("cds: cannot list artifacts in %s-%s-%s-%s/%s: not found", project, application, env, pipeline, tag)
	}
	if code >= 300 {
		return nil, fmt.Errorf("cds: cannot list artifacts in %s-%s-%s-%s/%s: %d", project, application, env, pipeline, tag, code)
	}

	var arts []Artifact
	err = json.Unmarshal(data, &arts)
	if err != nil {
		return nil, err
	}

	return arts, nil
}

// UploadArtifact read file at filePath and upload it in projet-pipeline-tag starage directory
func UploadArtifact(project string, pipeline string, application string, tag string, filePath string, buildNumber int, env string) (bool, time.Duration, error) {
	t0 := time.Now()
	tag = url.QueryEscape(tag)
	tag = strings.Replace(tag, "/", "-", -1)

	fileForMD5, errop := os.Open(filePath)
	if errop != nil {
		return false, 0, fmt.Errorf("unable on open file %s (%v)", filePath, errop)
	}

	//File stat
	stat, errst := fileForMD5.Stat()
	if errst != nil {
		return false, 0, fmt.Errorf("unable to get file info (%v)", errst)
	}

	//Compute md5sum
	hash := md5.New()
	if _, errcopy := io.Copy(hash, fileForMD5); errcopy != nil {
		return false, 0, fmt.Errorf("unable to read file content (%v)", errcopy)
	}
	hashInBytes := hash.Sum(nil)[:16]
	md5sumStr := hex.EncodeToString(hashInBytes)
	fileForMD5.Close()

	//Reopen the file because we already read it for md5
	fileContent, erro := ioutil.ReadFile(filePath)
	if erro != nil {
		return false, 0, fmt.Errorf("unable to read file %s (%v)", filePath, erro)
	}
	_, name := filepath.Split(filePath)

	bodyRes, _, _ := Request("GET", "/artifact/store", nil)
	if len(bodyRes) > 0 {
		store := new(ArtifactsStore)
		_ = json.Unmarshal(bodyRes, store)

		if store.TemporaryURLSupported {
			tempURL, dur, err := uploadArtifactWithTempURL(project, pipeline, application, env, tag, buildNumber, name, fileContent, stat, md5sumStr)
			if err == nil {
				return tempURL, dur, err // do not wrap error here, could be nil
			}
		}
	}

	// if we are here, upload with tempURL didn't work. Fallback to download with CDS API.
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, errc := writer.CreateFormFile(name, filepath.Base(filePath))
	if errc != nil {
		return false, 0, fmt.Errorf("unable to create multipart form file (%v)", errc)
	}

	if _, err := io.Copy(part, bytes.NewReader(fileContent)); err != nil {
		return false, 0, fmt.Errorf("unable to read file content (%v)", err)
	}

	writer.WriteField("env", env)
	writer.WriteField("size", strconv.FormatInt(stat.Size(), 10))
	writer.WriteField("perm", strconv.FormatUint(uint64(stat.Mode().Perm()), 10))
	writer.WriteField("md5sum", md5sumStr)

	if err := writer.Close(); err != nil {
		return false, 0, fmt.Errorf("unable to close multipart form writer (%v)", err)
	}

	var bodyReader io.Reader
	bodyReader = body

	var err error
	for i := 0; i < 10; i++ {
		var buf = new(bytes.Buffer)
		tee := io.TeeReader(bodyReader, buf)
		err = uploadArtifact(project, pipeline, application, tag, tee, name, writer.FormDataContentType(), buildNumber, env)
		if err == nil {
			return false, time.Since(t0), nil
		}
		time.Sleep(3 * time.Second)
		bodyReader = buf
	}

	return false, 0, fmt.Errorf("x10: %s", err)
}

func uploadArtifactWithTempURL(project, pipeline, application, env, tag string, buildNumber int, filename string, fileContent []byte, stat os.FileInfo, md5sum string) (bool, time.Duration, error) {
	t0 := time.Now()
	art := Artifact{
		Name:   filename,
		MD5sum: md5sum,
		Size:   stat.Size(),
		Perm:   uint32(stat.Mode().Perm()),
	}

	b, err := json.Marshal(art)
	if err != nil {
		return true, 0, err
	}

	path := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/%d/artifact/%s/url?envName=%s", project, application, pipeline, buildNumber, tag, url.QueryEscape(env))
	body, _, err := Request("POST", path, b)
	if err != nil {
		return true, 0, err
	}

	if err := json.Unmarshal(body, &art); err != nil {
		return true, 0, err
	}

	if verbose {
		fmt.Printf("Uploading %s with to %s", art.Name, art.TempURL)
	}

	//Post the file to the temporary URL
	req, errRequest := http.NewRequest("PUT", art.TempURL, bytes.NewReader(fileContent))
	if errRequest != nil {
		return true, 0, errRequest
	}

	resp, err := client.Do(req)
	if err != nil {
		return true, 0, err
	}

	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return true, 0, err
	}

	if resp.StatusCode >= 300 {
		return true, 0, fmt.Errorf("Unable to upload artifact: (HTTP %d) %s", resp.StatusCode, string(body))
	}

	//Call the API back to store the artifact in DB
	b, err = json.Marshal(art)
	if err != nil {
		return true, 0, err
	}

	//Try 50 times to make the callback
	const retry = 50
	var globalErr error
	for i := 0; i < retry; i++ {
		path = fmt.Sprintf("/project/%s/application/%s/pipeline/%s/%d/artifact/%s/url/callback?envName=%s", project, application, pipeline, buildNumber, tag, url.QueryEscape(env))
		_, _, globalErr = Request("POST", path, b)
		if globalErr == nil {
			return true, time.Since(t0), nil
		}
	}

	return true, time.Since(t0), globalErr
}

func uploadArtifact(project string, pipeline string, application string, tag string, body io.Reader, name string, contentType string, buildNumber int, env string) error {
	uri := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/%d/artifact/%s", project, application, pipeline, buildNumber, tag)
	_, code, err := UploadMultiPart("POST", uri, body,
		SetHeader(ArtifactFileName, name),
		SetHeader("Content-Type", contentType))
	if err != nil {
		return err
	}

	if code >= 300 {
		return fmt.Errorf("HTTP Error %d", code)
	}

	return nil
}
