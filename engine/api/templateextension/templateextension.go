package templateextension

import (
	"bytes"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/template"
	"github.com/spf13/viper"
)

//Get returns action plugin metadata and parameters list
func Get(name, path string) (*sdk.TemplateExtension, []sdk.TemplateParam, error) {
	//FIXME: run this in a jail with apparmor
	log.Debug("templateextension.Get> Getting info from '%s' (%s)", name, path)
	client := template.NewClient(name, path, "", "", true)
	defer func() {
		log.Debug("templateextension.Get> kill rpc-server")
		client.Kill()
	}()
	log.Debug("templateextension.Get> Client '%s'", name)
	_templ, err := client.Instance()
	if err != nil {
		return nil, nil, err
	}

	fi, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer fi.Close()
	stat, err := fi.Stat()
	if err != nil {
		return nil, nil, err
	}

	//Compute md5sum
	hash := md5.New()
	if _, err := io.Copy(hash, fi); err != nil {
		return nil, nil, err
	}
	hashInBytes := hash.Sum(nil)[:16]
	md5sumStr := hex.EncodeToString(hashInBytes)
	params := _templ.Parameters()
	te := sdk.TemplateExtension{
		Filename:    name,
		Name:        _templ.Name(),
		Type:        _templ.Type(),
		Author:      _templ.Author(),
		Description: _templ.Description(),
		Identifier:  _templ.Identifier(),
		Path:        path,
		Size:        stat.Size(),
		Perm:        uint32(stat.Mode().Perm()),
		MD5Sum:      md5sumStr,
		Params:      params,
		Actions:     _templ.ActionsNeeded(),
	}

	return &te, params, nil
}

//Instance returns the template instance
func Instance(tmpl *sdk.TemplateExtension, u *sdk.User, sessionKey sessionstore.SessionKey) (template.Interface, func(), error) {
	//Fetch fro mobject store
	buf, err := objectstore.FetchTemplateExtension(*tmpl)
	if err != nil {
		return nil, nil, err
	}

	//Read the buffer
	btes, err := ioutil.ReadAll(buf)
	if err != nil {
		return nil, nil, err
	}

	tmp, err := ioutil.TempDir("", "cds-template")
	if err != nil {
		log.Critical("Instance> %s", err)
		return nil, nil, err
	}
	deferFunc := func() {
		log.Debug("Instance> deleting file %s", tmp)
		os.RemoveAll(tmp)
	}

	log.Debug("Instance> creating temporary directory")
	tmpfn := filepath.Join(tmp, fmt.Sprintf("template-%d", tmpl.ID))
	f, err := os.OpenFile(tmpfn, os.O_WRONLY|os.O_CREATE, 0700)
	if err != nil {
		log.Critical("Instance> %s", err)
		return nil, deferFunc, err
	}

	if _, err := io.Copy(f, bytes.NewBuffer(btes)); err != nil {
		log.Critical("Instance> %s", err)
		return nil, deferFunc, err
	}
	f.Close()

	//The template will call local API
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "127.0.0.1"
	}

	//FIXME: export tls feature will impact this
	log.Debug("Instance>  %s:%s", u.Username, string(sessionKey))
	client := template.NewClient(tmpl.Name, f.Name(), u.Username+":"+string(sessionKey), "http://"+hostname+":"+viper.GetString("listen_port"), true)
	deferFunc = func() {
		client.Kill()
		os.RemoveAll(f.Name())
	}

	_templ, err := client.Instance()
	if err != nil {
		return nil, deferFunc, err
	}

	return _templ, deferFunc, nil
}

//Apply will call the apply function of the template and returns a fresh new application
func Apply(templ template.Interface, proj *sdk.Project, params []sdk.TemplateParam, appName string) (*sdk.Application, error) {
	regexp := regexp.MustCompile(sdk.NamePattern)
	if !regexp.MatchString(appName) {
		return nil, sdk.ErrInvalidApplicationPattern
	}

	parameters := map[string]string{}
	for _, p := range params {
		parameters[p.Name] = p.Value
	}
	templParameters := template.NewParameters(parameters)
	applyOptions := template.NewApplyOptions(proj.Key, appName, *templParameters)
	app, err := templ.Apply(applyOptions)

	app.Name = appName
	app.ProjectKey = proj.Key

	return &app, err
}

//All returns all template extensions
func All(db *sql.DB) ([]sdk.TemplateExtension, error) {
	dbmap := database.DBMap(db)
	tmpls := []database.TemplateExtension{}
	_, err := dbmap.Select(&tmpls, "select * from template order by id")
	if err != nil {
		log.Warning("All> Error: %s", err)
		return nil, err
	}

	sdktmpls := []sdk.TemplateExtension{}

	//Load actions and params
	for i := range tmpls {
		_, err := dbmap.Select(&tmpls[i].Actions, "select action.name from action, template_action where template_action.action_id = action.id and template_id = $1", tmpls[i].ID)
		if err != nil {
			log.Warning("All> Error: %s", err)
			return nil, err
		}
		params := []sdk.TemplateParam{}
		str, err := dbmap.SelectStr("select params from template_params where template_id = $1", tmpls[i].ID)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(str), &params); err != nil {
			return nil, err
		}
		tmpls[i].Params = params
		sdktmpls = append(sdktmpls, sdk.TemplateExtension(tmpls[i]))
	}
	return sdktmpls, nil
}

//LoadByID returns a templateextension from its ID
func LoadByID(db *sql.DB, id int64) (*sdk.TemplateExtension, error) {
	//Get the database map
	dbmap := database.DBMap(db)

	//Find it
	templ := database.TemplateExtension{}
	if err := dbmap.SelectOne(&templ, "select * from template where id = $1", id); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNotFound
		}
		log.Warning("deleteTemplateHandler>%T %s", err, err)
		return nil, err
	}

	sdktmpl := sdk.TemplateExtension(templ)
	return &sdktmpl, nil
}

//Insert inserts a new template
func Insert(db *sql.DB, sdktmpl *sdk.TemplateExtension) error {
	templ := database.TemplateExtension(*sdktmpl)
	//Get the database map
	dbmap := database.DBMap(db)
	if err := dbmap.Insert(&templ); err != nil {
		return err
	}
	sdktmpl.ID = templ.ID
	sdktmpl.Actions = templ.Actions
	sdktmpl.Params = templ.Params
	return nil
}

//Update updates the provided template given it ID
func Update(db *sql.DB, sdktmpl *sdk.TemplateExtension) error {
	templ := database.TemplateExtension(*sdktmpl)
	//Get the database map
	dbmap := database.DBMap(db)
	_, err := dbmap.Update(&templ)
	sdktmpl.Actions = templ.Actions
	sdktmpl.Params = templ.Params
	return err
}

//Delete deletes the provided template given it ID
func Delete(db *sql.DB, sdktmpl *sdk.TemplateExtension) error {
	templ := database.TemplateExtension(*sdktmpl)
	//Get the database map
	dbmap := database.DBMap(db)
	n, err := dbmap.Delete(&templ)
	if n == 0 {
		return sdk.ErrNotFound
	}
	return err
}

//LoadByName returns a templateextension from its name
func LoadByName(db *sql.DB, name string) (*sdk.TemplateExtension, error) {
	log.Debug("Loading template %s", name)
	// Get template from DB
	dbmap := database.DBMap(db)
	tmpl := database.TemplateExtension{}
	if err := dbmap.SelectOne(&tmpl, "select * from template where name = $1", name); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrUnknownTemplate
		}
		return nil, err
	}

	// Load the template binary
	sdktmpl := sdk.TemplateExtension(tmpl)
	return &sdktmpl, nil
}

const UglyID = 10000

var EmptyTemplate = sdk.Template{
	ID:          UglyID,
	Name:        "Void",
	Description: "Empty template",
}

//LoadByType returns list of templates by type
func LoadByType(db *sql.DB, t string) ([]sdk.Template, error) {
	var tpl []sdk.Template
	tpl = []sdk.Template{
		EmptyTemplate,
	}

	tplFromDB := []sdk.TemplateExtension{}
	dbmap := database.DBMap(db)
	if _, err := dbmap.Select(&tplFromDB, "select * from template where type = $1 order by name", t); err != nil {
		log.Warning("getTypedTemplatesHandler> Error : %s", err)
		return nil, err
	}

	for _, t := range tplFromDB {
		params := []sdk.TemplateParam{}
		str, err := dbmap.SelectStr("select params from template_params where template_id = $1", t.ID)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(str), &params); err != nil {
			return nil, err
		}

		tpl = append(tpl, sdk.Template{
			ID:          t.ID,
			Name:        t.Name,
			Description: t.Description,
			Params:      params,
		})
	}

	return tpl, nil
}
