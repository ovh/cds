package sdk

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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
	ID          int64  `json:"id"`
	Project     string `json:"project"`
	Pipeline    string `json:"pipeline"`
	Application string `json:"application"`
	Environment string `json:"environment"`
	BuildNumber int    `json:"build_number"`
	Name        string `json:"name"`
	Tag         string `json:"tag"`

	DownloadHash string `json:"download_hash"`
	Size         int64  `json:"size,omitempty"`
	Perm         uint32 `json:"perm,omitempty"`
	MD5sum       string `json:"md5sum,omitempty"`
	ObjectPath   string `json:"object_path,omitempty"`
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

	uri := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/artifact/%s?envName=%s", project, application, pipeline, tag, env)
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

	var err error
	for i := 0; i < 5; i++ {
		err = uploadArtifact(project, pipeline, application, tag, filePath, buildNumber, env)
		if err == nil {
			return nil
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("x5: %s", err)
}

func uploadArtifact(project string, pipeline string, application string, tag string, filePath string, buildNumber int, env string) error {
	uri := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/%d/artifact/%s", project, application, pipeline, buildNumber, tag)

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}

	//File stat
	stat, err := file.Stat()
	if err != nil {
		return err
	}

	//Compute md5sum
	hash := md5.New()
	if _, errcopy := io.Copy(hash, file); errcopy != nil {
		return errcopy
	}
	hashInBytes := hash.Sum(nil)[:16]
	md5sumStr := hex.EncodeToString(hashInBytes)
	file.Close()

	//Reopen the file because we already read it for md5
	file, err = os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, name := filepath.Split(filePath)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(name, filepath.Base(filePath))
	if err != nil {
		return err
	}
	_, err = io.Copy(part, file)

	writer.WriteField("env", env)
	writer.WriteField("size", strconv.FormatInt(stat.Size(), 10))
	writer.WriteField("perm", strconv.FormatUint(uint64(stat.Mode().Perm()), 10))
	writer.WriteField("md5sum", md5sumStr)

	if errclose := writer.Close(); errclose != nil {
		return errclose
	}

	_, code, err := UploadMultiPart("POST", uri, body, SetHeader(ArtifactFileName, name), SetHeader("Content-Type", writer.FormDataContentType()))
	if err != nil {
		return err
	}

	if code >= 300 {
		return fmt.Errorf("HTTP Error %d\n", code)
	}

	return nil
}
