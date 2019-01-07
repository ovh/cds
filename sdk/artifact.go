package sdk

import (
	"fmt"
	"net/url"
	"strings"
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
	MD5sum           string `json:"md5sum,omitempty" cli:"-"`
	SHA512sum        string `json:"sha512sum,omitempty" cli:"sha512sum"`
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
	ServeStaticFiles = "Serve Static Files"
)
