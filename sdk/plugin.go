package sdk

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
)

//DownloadPlugin download plugin from action
func DownloadPlugin(name string, destdir string) error {
	var lasterr error
	for retry := 5; retry >= 0; retry-- {
		uri := fmt.Sprintf("/plugin/download/%s?accept-redirect=true", name)
		reader, code, err := Stream("GET", uri, nil)
		if err != nil {
			lasterr = err
			continue
		}
		if code >= 300 {
			lasterr = fmt.Errorf("HTTP %d", code)
			continue
		}
		destPath := path.Join(destdir, name)
		//If the file already exists, remove it
		if _, errstat := os.Stat(destPath); errstat == nil {
			os.RemoveAll(destPath)
		}

		f, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}

		if _, err = io.Copy(f, reader); err != nil {
			lasterr = err
		}

		if err := f.Close(); err == nil {
			return nil
		}
	}
	return fmt.Errorf("x5: %s", lasterr)
}

//UploadPlugin uploads binary file to perform a new action
func UploadPlugin(filePath string, update bool) ([]byte, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, err
	}

	file, erro := os.Open(filePath)
	if erro != nil {
		return nil, erro
	}
	defer file.Close()

	//_, name := filepath.Split(filePath)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, errc := writer.CreateFormFile("UploadFile", filepath.Base(filePath))
	if errc != nil {
		return nil, errc
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}
	method := "POST"
	if update {
		method = "PUT"
	}
	btes, code, err := UploadMultiPart(method, "/plugin", body, SetHeader("uploadfile", filePath), SetHeader("Content-Type", writer.FormDataContentType()))
	if err != nil {
		return nil, err
	}

	if code >= 300 {
		return nil, fmt.Errorf("HTTP Error %d", code)
	}

	return btes, nil
}

//DeletePlugin delete plugin
func DeletePlugin(name string) error {
	path := fmt.Sprintf("/plugin/%s", name)

	_, _, err := Request("DELETE", path, nil)
	if err != nil {
		return err
	}

	return nil
}

// These are type of plugins
const (
	GRPCPluginDeploymentPlatform = "deployment_platform"
	GRPCPluginAction             = "action"
)

// GRPCPlugin is the type representing a plugin over GRPC
type GRPCPlugin struct {
	ID          int64              `json:"id" cli:"id" db:"id"`
	Name        string             `json:"name" cli:"name,key" db:"name"`
	Type        string             `json:"type" cli:"type" db:"type"`
	Author      string             `json:"author" cli:"author" db:"author"`
	Description string             `json:"description" cli:"description" db:"description"`
	Binaries    []GRPCPluginBinary `json:"binaries" cli:"-" db:"-"`
}

// GetBinary returns the binary for a specific os and arch
func (p GRPCPlugin) GetBinary(os, arch string) *GRPCPluginBinary {
	for _, b := range p.Binaries {
		if b.OS == os && b.Arch == arch {
			return &b
		}
	}
	return nil
}

// GRPCPluginBinary represents a binary file (for a specific os and arch) serving a GRPCPlugin
type GRPCPluginBinary struct {
	OS               string          `json:"os,omitempty" yaml:"os"`
	Arch             string          `json:"arch,omitempty" yaml:"arch"`
	Name             string          `json:"name,omitempty" yaml:"-"`
	ObjectPath       string          `json:"object_path,omitempty" yaml:"-"`
	Size             int64           `json:"size,omitempty" yaml:"-"`
	Perm             uint32          `json:"perm,omitempty" yaml:"-"`
	MD5sum           string          `json:"md5sum,omitempty" yaml:"-"`
	SHA512sum        string          `json:"sha512sum,omitempty" yaml:"-"`
	TempURL          string          `json:"temp_url,omitempty" yaml:"-"`
	TempURLSecretKey string          `json:"-" yaml:"-"`
	Cmd              string          `json:"cmd,omitempty" yaml:"cmd"`
	Args             []string        `json:"args,omitempty" yaml:"args"`
	Requirements     RequirementList `json:"requirements,omitempty" yaml:"requirements"`
	FileContent      []byte          `json:"file_content,omitempty" yaml:"-"` //only used for upload
	PluginName       string          `json:"plugin_name,omitempty" yaml:"-"`
}

// GetName is a part of the objectstore.Object interface implementation
func (b GRPCPluginBinary) GetName() string {
	return b.Name
}

// GetPath is a part of the objectstore.Object interface implementation
func (b GRPCPluginBinary) GetPath() string {
	return b.Name + "-" + b.OS + "-" + b.Arch
}
