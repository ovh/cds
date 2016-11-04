package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"

	"gopkg.in/gorp.v1"
)

// TemplateParam can be a String/Date/Script/URL...
type TemplateParam struct {
	ID          int64        `json:"id" yaml:"-"`
	Name        string       `json:"name"`
	Type        VariableType `json:"type"`
	Value       string       `json:"value"`
	Description string       `json:"description" yaml:"desc,omitempty"`
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

// ApplyApplicationTemplates creates given application and apply build and deployment templates
func ApplyApplicationTemplates(projectKey string, name, repo string, build, deploy Template) (*Application, error) {
	uri := fmt.Sprintf("/template/%s", projectKey)

	app := &Application{
		Name: name,
		//		BuildTemplate:  build,
		//		DeployTemplate: deploy,
		Variable: []Variable{
			Variable{
				Name:  "repo",
				Type:  StringVariable,
				Value: repo,
			},
		},
	}

	data, err := json.Marshal(app)
	if err != nil {
		return nil, err
	}

	data, code, err := Request("POST", uri, data)
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("HTTP %d", code)
	}

	err = json.Unmarshal(data, app)
	if err != nil {
		return nil, err
	}

	return app, nil
}

//TemplateExtention represents a template store as a binary extension
type TemplateExtention struct {
	ID          int64           `json:"id" db:"id"`
	Name        string          `json:"name" db:"name"`
	Type        string          `json:"type" db:"type"`
	Author      string          `json:"author" db:"author"`
	Description string          `json:"description"`
	Identifier  string          `json:"identifier" db:"identifier"`
	Size        int64           `json:"-" db:"size"`
	Perm        uint32          `json:"-" db:"perm"`
	MD5Sum      string          `json:"md5sum" db:"md5sum"`
	ObjectPath  string          `json:"-" db:"object_path"`
	Filename    string          `json:"-" db:"-"`
	Path        string          `json:"-" db:"-"`
	Params      []TemplateParam `json:"-" db:"-"`
}

//PostInsert is a DB Hook on TemplateExtention to store params as JSON in DB
func (t *TemplateExtention) PostInsert(s gorp.SqlExecutor) error {
	btes, err := json.Marshal(t.Params)
	if err != nil {
		return err
	}

	query := "insert into template_params (template_id, params) values ($1, $2)"
	if _, err := s.Exec(query, t.ID, btes); err != nil {
		return err
	}
	return nil
}

//PostUpdate is a DB Hook on TemplateExtention to store params as JSON in DB
func (t *TemplateExtention) PostUpdate(s gorp.SqlExecutor) error {
	btes, err := json.Marshal(t.Params)
	if err != nil {
		return err
	}

	query := "update template_params set params = $2 where template_id = $1"
	if _, err := s.Exec(query, t.ID, btes); err != nil {
		return err
	}
	return nil
}

//PreDelete is a DB Hook on TemplateExtention to store params as JSON in DB
func (t *TemplateExtention) PreDelete(s gorp.SqlExecutor) error {
	query := "delete from template_params where template_id = $1"
	if _, err := s.Exec(query, t.ID); err != nil {
		return err
	}
	return nil

}

//GetName returns the name of the template extension
func (a *TemplateExtention) GetName() string {
	return a.Name
}

//GetPath returns the storage path of the template extension
func (a *TemplateExtention) GetPath() string {
	return fmt.Sprintf("templates")
}

//DownloadTemplate download Template from action
func DownloadTemplate(name string, destdir string) error {
	var lasterr error
	for retry := 5; retry >= 0; retry-- {
		uri := fmt.Sprintf("/Template/download/%s", name)
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
		if _, err := os.Stat(destPath); err == nil {
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
			fmt.Printf("Download %s completed\n", destPath)
			return nil
		}
	}

	return fmt.Errorf("x5: %s", lasterr)
}

//UploadTemplate uploads binary file to perform a new action
func UploadTemplate(filePath string, update bool, name string) ([]byte, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("UploadFile", filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)

	err = writer.Close()
	if err != nil {
		return nil, err
	}
	path := "/template/add"
	method := "POST"
	if update {
		method = "PUT"
		path = "/template/"

		btes, _, err := Request("GET", "/template", nil)
		if err != nil {
			return nil, err
		}
		tmpls := []TemplateExtention{}
		if err := json.Unmarshal(btes, &tmpls); err != nil {
			return nil, err
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
	}
	btes, code, err := UploadMultiPart(method, path, body, SetHeader("uploadfile", filePath), SetHeader("Content-Type", writer.FormDataContentType()))
	if err != nil {
		return nil, err
	}

	if code >= 300 {
		return nil, fmt.Errorf("HTTP Error %d\n", code)
	}

	return btes, nil
}

//DeleteTemplate delete Template
func DeleteTemplate(name string) error {
	path := fmt.Sprintf("/Template/%s", name)

	_, _, err := Request("DELETE", path, nil)
	if err != nil {
		return err
	}

	return nil
}
