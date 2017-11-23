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
	TempURLSecretKey string `json:"temp_url_secret_key,omitempty"`
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

	for retry := 5; retry >= 0; retry-- {
		uri := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/artifact/download/%d", project, app, pip, a.ID)
		reader, code, err := Stream("GET", uri, nil)
		if err != nil {
			lasterr = err
			continue
		}
		if code >= 300 {
			lasterr = fmt.Errorf("HTTP %d", code)
			continue
		}
		destPath := path.Join(destdir, a.Name)

		mode := os.FileMode(0644)
		if a.Perm != uint32(0) {
			mode = os.FileMode(a.Perm)
		}

		f, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY, mode)
		if err != nil {
			lasterr = err
			continue
		}

		_, err = io.Copy(f, reader)
		if err != nil {
			lasterr = err
		}

		f.Close()
		if err == nil {
			return nil
		}
	}

	return fmt.Errorf("x5: %s", lasterr)
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
func UploadArtifact(project string, pipeline string, application string, tag string, filePath string, buildNumber int, env string) error {

	tag = url.QueryEscape(tag)
	tag = strings.Replace(tag, "/", "-", -1)

	fileForMD5, errop := os.Open(filePath)
	if errop != nil {
		return errop
	}

	//File stat
	stat, errst := fileForMD5.Stat()
	if errst != nil {
		return errst
	}

	//Compute md5sum
	hash := md5.New()
	if _, errcopy := io.Copy(hash, fileForMD5); errcopy != nil {
		return errcopy
	}
	hashInBytes := hash.Sum(nil)[:16]
	md5sumStr := hex.EncodeToString(hashInBytes)
	fileForMD5.Close()

	//Reopen the file because we already read it for md5
	fileReopen, erro := os.Open(filePath)
	if erro != nil {
		return erro
	}
	defer fileReopen.Close()
	_, name := filepath.Split(filePath)

	bodyRes, _, _ := Request("GET", "/artifact/store", nil)
	if len(bodyRes) > 0 {
		store := new(ArtifactsStore)
		_ = json.Unmarshal(bodyRes, store)

		if store.TemporaryURLSupported {
			return uploadArtifactWithTempURL(project, pipeline, application, env, tag, buildNumber, name, fileReopen, stat, md5sumStr)
		}
	}

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, errc := writer.CreateFormFile(name, filepath.Base(filePath))
	if errc != nil {
		return errc
	}

	if _, err := io.Copy(part, fileReopen); err != nil {
		return err
	}

	writer.WriteField("env", env)
	writer.WriteField("size", strconv.FormatInt(stat.Size(), 10))
	writer.WriteField("perm", strconv.FormatUint(uint64(stat.Mode().Perm()), 10))
	writer.WriteField("md5sum", md5sumStr)

	if err := writer.Close(); err != nil {
		return err
	}

	var bodyReader io.Reader
	bodyReader = body

	var err error
	for i := 0; i < 10; i++ {
		var buf = new(bytes.Buffer)
		tee := io.TeeReader(bodyReader, buf)
		err = uploadArtifact(project, pipeline, application, tag, tee, name, writer.FormDataContentType(), buildNumber, env)
		if err == nil {
			return nil
		}
		time.Sleep(3 * time.Second)
		bodyReader = buf
	}

	return fmt.Errorf("x10: %s", err)
}

func uploadArtifactWithTempURL(project, pipeline, application, env, tag string, buildNumber int, filename string, file io.Reader, stat os.FileInfo, md5sum string) error {

	art := Artifact{
		Name:   filename,
		MD5sum: md5sum,
		Size:   stat.Size(),
		Perm:   uint32(stat.Mode().Perm()),
	}

	b, err := json.Marshal(art)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/%d/artifact/%s/url", project, application, pipeline, buildNumber, tag)
	body, _, err := Request("POST", path, b)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, &art); err != nil {
		return err
	}

	fmt.Println("Temprary URL: ", art.TempURL)

	//Post the file to the temporary URL
	req, errRequest := http.NewRequest("PUT", art.TempURL, file)
	if errRequest != nil {
		return errRequest
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 300 {
		return fmt.Errorf("Unable to upload artifact: (HTTP %d) %s", resp.StatusCode, string(body))
	}

	//Call the API back to store the artifact in DB
	b, err = json.Marshal(art)
	if err != nil {
		return err
	}
	path = fmt.Sprintf("/project/%s/application/%s/pipeline/%s/%d/artifact/%s/url/callback", project, application, pipeline, buildNumber, tag)
	if _, _, err := Request("POST", path, b); err != nil {
		return err
	}

	return nil
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
