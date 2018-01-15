package actionplugin

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/plugin"
)

//Get returns action plugin metadata and parameters list
func Get(name, path string) (*sdk.ActionPlugin, *plugin.Parameters, error) {
	//FIXME: run this in a jail with apparmor
	log.Debug("actionplugin.Get> Getting info from '%s' (%s)", name, path)
	client := plugin.NewClient(context.Background(), name, path, "ID", "http://127.0.0.1:8081", true)
	defer func() {
		log.Debug("actionplugin.Get> kill rpc-server")
		client.Kill()
	}()
	log.Debug("actionplugin.Get> Client '%s'", name)
	_plugin, err := client.Instance()
	if err != nil {
		return nil, nil, sdk.WrapError(err, "actionplugin.Get> ")
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

	ap := sdk.ActionPlugin{
		Filename:    name,
		Name:        _plugin.Name(),
		Author:      _plugin.Author(),
		Description: _plugin.Description(),
		Path:        path,
		Size:        stat.Size(),
		Perm:        uint32(stat.Mode().Perm()),
		MD5sum:      md5sumStr,
	}

	params := _plugin.Parameters()

	return &ap, &params, nil
}

func actionPluginToAction(ap *sdk.ActionPlugin, params *plugin.Parameters) (*sdk.Action, error) {
	actionParams := []sdk.Parameter{}
	names := params.Names()
	for _, p := range names {
		var t string
		switch params.GetType(p) {
		case plugin.EnvironmentParameter:
			t = sdk.EnvironmentParameter
		case plugin.PipelineParameter:
			t = sdk.PipelineParameter
		case plugin.ListParameter:
			t = sdk.ListParameter
		case plugin.NumberParameter:
			t = sdk.NumberParameter
			//	case plugin.PasswordParameter:
			//		t = sdk.PasswordParameter
		case plugin.StringParameter:
			t = sdk.StringParameter
		case plugin.TextParameter:
			t = sdk.TextParameter
		case plugin.BooleanParameter:
			t = sdk.BooleanParameter
		default:
			log.Warning("plugin.Insert> unsupported parameter type '%s'", t)
			return nil, fmt.Errorf("Unsupported parameter type '%s' in plugin '%s'", t, ap.Name)
		}

		actionParams = append(actionParams, sdk.Parameter{
			Name:        p,
			Value:       params.GetValue(p),
			Type:        t,
			Description: params.GetDescription(p),
		})
	}

	a := &sdk.Action{
		Name:        ap.Name,
		Type:        sdk.PluginAction,
		Description: ap.Description,
		Requirements: sdk.RequirementList{
			sdk.Requirement{
				Name:  ap.Name,
				Type:  sdk.PluginRequirement,
				Value: ap.Name,
			},
		},
		Parameters: actionParams,
		Enabled:    true,
	}
	return a, nil
}

//Insert create action in database
func Insert(db gorp.SqlExecutor, ap *sdk.ActionPlugin, params *plugin.Parameters) (*sdk.Action, error) {
	a, err := actionPluginToAction(ap, params)
	if err != nil {
		return nil, err
	}

	if err = action.InsertAction(db, a, true); err != nil {
		log.Warning("plugin.Insert> Action: Cannot insert action: %s\n", err)
		return nil, err
	}

	query := `INSERT INTO plugin (name, size, perm, md5sum, object_path) VALUES ($1, $2, $3, $4, $5) RETURNING id`
	if err = db.QueryRow(query, ap.Name, ap.Size, ap.Perm, ap.MD5sum, ap.ObjectPath).Scan(&ap.ID); err != nil {
		return nil, err
	}
	return a, nil
}

//Update action in database
func Update(db gorp.SqlExecutor, ap *sdk.ActionPlugin, params *plugin.Parameters, userID int64) (*sdk.Action, error) {
	a, err := actionPluginToAction(ap, params)
	if err != nil {
		return nil, err
	}

	//oldA, err := action.LoadPublicAction(db, a.Name, action.WithClearPasswords())
	oldA, err := action.LoadPublicAction(db, a.Name)
	if err != nil {
		return nil, err
	}
	a.ID = oldA.ID

	if err := action.UpdateActionDB(db, a, userID); err != nil {
		return nil, err
	}

	query := "DELETE FROM plugin WHERE name = $1"
	if _, err := db.Exec(query, a.Name); err != nil {
		return nil, err
	}

	query = `INSERT INTO plugin (name, size, perm, md5sum, object_path) VALUES ($1, $2, $3, $4, $5) RETURNING id`
	if err = db.QueryRow(query, ap.Name, ap.Size, ap.Perm, ap.MD5sum, ap.ObjectPath).Scan(&ap.ID); err != nil {
		return nil, err
	}
	return a, nil
}

//Delete action in database
func Delete(db *gorp.DbMap, name string, userID int64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	a, err := action.LoadPublicAction(tx, name)
	if err != nil {
		return sdk.WrapError(err, "plugin.Delete> Action: Cannot get action %s", name)
	}

	query := "DELETE FROM plugin WHERE name = $1"
	if _, err := tx.Exec(query, a.Name); err != nil {
		return err
	}

	if err := action.DeleteAction(tx, a.ID, userID); err != nil {
		return err
	}

	return tx.Commit()

}
