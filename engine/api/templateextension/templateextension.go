package templateextension

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/template"
	"github.com/spf13/viper"
)

//Get returns action plugin metadata and parameters list
func Get(name, path string) (*sdk.TemplateExtention, []sdk.TemplateParam, error) {
	//FIXME: run this in a jail with apparmor
	log.Debug("templateextension.Get> Getting info from '%s' (%s)", name, path)
	client := template.NewClient(name, path, "ID", "http://127.0.0.1:8081", true)
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
	te := sdk.TemplateExtention{
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
	}

	return &te, params, nil
}

//Instance returns the template instance
func Instance(authDriver auth.Driver, tmpl *sdk.TemplateExtention, u *sdk.User) (template.Interface, func(), error) {
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

	//Create a session for current user
	sessionKey, err := auth.NewSession(authDriver, u)
	if err != nil {
		log.Critical("Instance> Error while creating new session: %s\n", err)
	}

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
