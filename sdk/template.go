package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
)

// TemplateParam can be a String/Date/Script/URL...
type TemplateParam struct {
	ID          int64  `json:"id" yaml:"-"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Value       string `json:"value"`
	Description string `json:"description" yaml:"desc,omitempty"`
}

// Template definition to help users bootstrap their pipelines
type Template struct {
	ID          int64           `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Params      []TemplateParam `json:"params"`
	Hook        bool            `json:"hook"`
}

// GetBuildTemplate Get the build template corresponding to the given name
func GetBuildTemplate(name string) (*Template, error) {
	tpls, err := GetBuildTemplates()
	if err != nil {
		return nil, err
	}

	for _, t := range tpls {
		if t.Name == name {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("%s: not found", err)
}

// GetBuildTemplates retrieves all existing build template from API
func GetBuildTemplates() ([]Template, error) {
	uri := "/template/build"

	data, code, err := Request("GET", uri, nil)
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("HTTP %d", code)
	}

	var tmpls []Template
	err = json.Unmarshal(data, &tmpls)
	if err != nil {
		return nil, err
	}

	return tmpls, nil
}

// GetDeploymentTemplates retrieves all existing deployment template from API
func GetDeploymentTemplates() ([]Template, error) {
	uri := "/template/deploy"

	data, code, err := Request("GET", uri, nil)
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("HTTP %d", code)
	}

	var tmpls []Template
	err = json.Unmarshal(data, &tmpls)
	if err != nil {
		return nil, err
	}

	return tmpls, nil
}

// ApplyApplicationTemplate creates given application and apply build template
func ApplyApplicationTemplate(projectKey string, name string, build Template) (*Application, error) {

	uri := fmt.Sprintf("/project/%s/template", projectKey)

	opts := ApplyTemplatesOptions{
		ApplicationName: name,
		TemplateName:    build.Name,
		TemplateParams:  build.Params,
	}

	btes, _ := json.Marshal(opts)
	data, code, err := Request("POST", uri, btes)
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, DecodeError(data)
	}

	app, err := GetApplication(projectKey, name)
	if err != nil {
		return nil, err
	}

	return app, nil
}

//TemplateExtension represents a template store as a binary extension
type TemplateExtension struct {
	ID          int64           `json:"id" db:"id"`
	Name        string          `json:"name" db:"name"`
	Type        string          `json:"type" db:"type"`
	Author      string          `json:"author" db:"author"`
	Description string          `json:"description" db:"description"`
	Identifier  string          `json:"identifier" db:"identifier"`
	Size        int64           `json:"-" db:"size"`
	Perm        uint32          `json:"-" db:"perm"`
	MD5Sum      string          `json:"md5sum" db:"md5sum"`
	ObjectPath  string          `json:"-" db:"object_path"`
	Filename    string          `json:"-" db:"-"`
	Path        string          `json:"-" db:"-"`
	Params      []TemplateParam `json:"params" db:"-"`
	Actions     []string        `json:"actions" db:"-"`
}

//ApplyTemplatesOptions represents arguments to create an application and all its components from templates
type ApplyTemplatesOptions struct {
	ApplicationName string          `json:"name"`
	TemplateName    string          `json:"template"`
	TemplateParams  []TemplateParam `json:"template_params"`
}

//GetName returns the name of the template extension
func (a *TemplateExtension) GetName() string {
	return a.Name
}

//GetPath returns the storage path of the template extension
func (a *TemplateExtension) GetPath() string {
	return fmt.Sprintf("templates")
}

//UploadTemplate uploads binary file to perform a new action
func UploadTemplate(filePath string, update bool, name string) ([]byte, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, err
	}

	file, erro := os.Open(filePath)
	if erro != nil {
		return nil, erro
	}
	defer file.Close()

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
	path := "/template/add"
	method := "POST"
	if update {
		method = "PUT"
		path = "/template/"

		if verbose {
			log.Println("Getting templates list")
		}

		btes, _, errrequest := Request("GET", "/template", nil)
		if errrequest != nil {
			return nil, errrequest
		}
		tmpls := []TemplateExtension{}
		if errjson := json.Unmarshal(btes, &tmpls); errjson != nil {
			return nil, errjson
		}

		if verbose {
			log.Println("Getting templates list : OK")
		}

		var found bool
		for _, t := range tmpls {
			if t.Name == name {
				path += fmt.Sprintf("%d", t.ID)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("Template %s not found", name)
		}

		if verbose {
			log.Printf("Found template at %s\n", path)
		}

	}
	btes, code, err := UploadMultiPart(method, path, body, SetHeader("uploadfile", filePath), SetHeader("Content-Type", writer.FormDataContentType()))
	if err != nil {
		return nil, err
	}

	if code >= 300 {
		return nil, fmt.Errorf("HTTP Error %d", code)
	}

	return btes, nil
}

//DeleteTemplate delete Template
func DeleteTemplate(name string) error {
	tmpls, err := ListTemplates()
	if err != nil {
		return err
	}

	var id int64
	var found bool
	for _, t := range tmpls {
		if t.Name == name {
			id = t.ID
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("Unable to found template %s", name)
	}

	path := fmt.Sprintf("/template/%d", id)
	if _, _, err := Request("DELETE", path, nil); err != nil {
		return err
	}

	return nil
}

//ListTemplates returns all templates
func ListTemplates() ([]TemplateExtension, error) {
	tmpls := []TemplateExtension{}
	body, code, err := Request("GET", "/template", nil)
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("HTTP Error %d", code)
	}
	if err := json.Unmarshal(body, &tmpls); err != nil {
		return nil, err
	}
	return tmpls, nil
}
