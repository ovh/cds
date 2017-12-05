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
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/template"
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
func Instance(tmpl *sdk.TemplateExtension, u *sdk.User, sessionKey sessionstore.SessionKey, apiURL string) (template.Interface, func(), error) {
	//Fetch fro mobject store
	buf, errf := objectstore.FetchTemplateExtension(*tmpl)
	if errf != nil {
		return nil, nil, errf
	}

	//Read the buffer
	btes, errr := ioutil.ReadAll(buf)
	if errr != nil {
		return nil, nil, errr
	}

	tmp, errt := ioutil.TempDir("", "cds-template")
	if errt != nil {
		log.Error("Instance> %s", errt)
		return nil, nil, errt
	}
	deferFunc := func() {
		log.Debug("Instance> deleting file %s", tmp)
		os.RemoveAll(tmp)
	}

	log.Debug("Instance> creating temporary directory")
	tmpfn := filepath.Join(tmp, fmt.Sprintf("template-%d", tmpl.ID))
	f, erro := os.OpenFile(tmpfn, os.O_WRONLY|os.O_CREATE, 0700)
	if erro != nil {
		log.Error("Instance> %s", erro)
		return nil, deferFunc, erro
	}

	if _, err := io.Copy(f, bytes.NewBuffer(btes)); err != nil {
		log.Error("Instance> %s", err)
		return nil, deferFunc, err
	}
	f.Close()

	//FIXME: export tls feature will impact this
	log.Debug("Instance>  %s:%s", u.Username, string(sessionKey))
	client := template.NewClient(tmpl.Name, f.Name(), u.Username+":"+string(sessionKey), apiURL, true)
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
func Apply(db gorp.SqlExecutor, store cache.Store, templ template.Interface, proj *sdk.Project, params []sdk.TemplateParam, appName string) (*sdk.Application, error) {
	regexp := sdk.NamePatternRegex
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

	// Check repo parameter
	for _, p := range params {
		if p.Type == sdk.RepositoryVariable {
			repoDatas := strings.SplitN(p.Value, "##", 2)

			// If repo from repository manager
			if len(repoDatas) == 2 {
				app.VCSServer = repoDatas[0]
				app.RepositoryFullname = repoDatas[1]

				var vcsServer *sdk.ProjectVCSServer
				for _, v := range proj.VCSServers {
					if v.Name == repoDatas[0] {
						vcsServer = &v
						break
					}
				}

				if vcsServer == nil {
					return nil, fmt.Errorf("Repomanager not found")
				}

				// overwrite application variable value with  correct URL
				for i := range app.Variable {
					v := &app.Variable[i]
					if v.Name == p.Name {
						client, errClient := repositoriesmanager.AuthorizedClient(db, store, vcsServer)
						if errClient != nil {
							log.Warning("ApplyTemplate> Cannot get client got %s %s : %s", proj.Key, app.VCSServer, errClient)
							return nil, errClient
						}
						appRepo, errRepo := client.RepoByFullname(app.RepositoryFullname)
						if errRepo != nil {
							log.Warning("ApplyTemplate> Cannot get repo by fullname %s : %s", app.RepositoryFullname, errRepo)
							return nil, errRepo
						}
						v.Value = appRepo.SSHCloneURL
					}
				}
			}
		}
	}
	app.Name = appName
	app.ProjectKey = proj.Key

	return &app, err
}

//All returns all template extensions
func All(dbmap *gorp.DbMap) ([]sdk.TemplateExtension, error) {
	tmpls := []TemplateExtension{}
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
func LoadByID(dbmap *gorp.DbMap, id int64) (*sdk.TemplateExtension, error) {
	//Find it
	templ := TemplateExtension{}
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
func Insert(dbmap *gorp.DbMap, sdktmpl *sdk.TemplateExtension) error {
	templ := TemplateExtension(*sdktmpl)
	//Get the database map
	if err := dbmap.Insert(&templ); err != nil {
		return err
	}
	sdktmpl.ID = templ.ID
	sdktmpl.Actions = templ.Actions
	sdktmpl.Params = templ.Params
	return nil
}

//Update updates the provided template given it ID
func Update(dbmap *gorp.DbMap, sdktmpl *sdk.TemplateExtension) error {
	templ := TemplateExtension(*sdktmpl)
	//Get the database map
	_, err := dbmap.Update(&templ)
	sdktmpl.Actions = templ.Actions
	sdktmpl.Params = templ.Params
	return err
}

//Delete deletes the provided template given it ID
func Delete(dbmap *gorp.DbMap, sdktmpl *sdk.TemplateExtension) error {
	templ := TemplateExtension(*sdktmpl)
	//Get the database map
	n, err := dbmap.Delete(&templ)
	if n == 0 {
		return sdk.ErrNotFound
	}
	return err
}

//LoadByName returns a templateextension from its name
func LoadByName(dbmap gorp.SqlExecutor, name string) (*sdk.TemplateExtension, error) {
	log.Debug("Loading template %s", name)
	// Get template from DB
	tmpl := TemplateExtension{}
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
func LoadByType(dbmap *gorp.DbMap, t string) ([]sdk.Template, error) {
	var tpl []sdk.Template
	tpl = []sdk.Template{
		EmptyTemplate,
	}

	tplFromDB := []sdk.TemplateExtension{}
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
