package templateextension

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"

	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/template"
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

	te := sdk.TemplateExtention{
		Filename:    name,
		Name:        _templ.Name(),
		Author:      _templ.Author(),
		Description: _templ.Description(),
		Identifier:  _templ.Identifier(),
		Path:        path,
		Size:        stat.Size(),
		Perm:        uint32(stat.Mode().Perm()),
		MD5Sum:      md5sumStr,
	}

	params := _templ.Parameters()

	return &te, params, nil
}
