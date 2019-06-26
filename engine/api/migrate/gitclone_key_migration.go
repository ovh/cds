package migrate

import (
	"regexp"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func GitClonePrivateKeyParameter(db *gorp.DbMap, store cache.Store) error {
	store.Publish(sdk.MaintenanceQueueName, "true")
	defer store.Publish(sdk.MaintenanceQueueName, "false")

	params, err := getprivateKeyActionParameters(db)
	if err != nil {
		return sdk.WrapError(err, "cannot get private key action parameters")
	}

	var globalError error
	for _, param := range params {
		newValue := getNewPrivateKeyValue(param.Value)
		if param.Value == "" {
			if _, err := db.Exec("UPDATE action_parameter SET type = 'ssh-key' WHERE id = $1", param.ID); err != nil {
				log.Error("cannot update action_parameter type %d %s : %v", param.ID, param.Name, err)
				globalError = sdk.WrapError(err, "%v", globalError)
			}
			continue
		}
		if newValue == param.Value {
			log.Warning("bad key, cannot get new private key value for param %d %s --> Value = %s", param.ID, param.Name, param.Value)
			continue
		}
		if _, err := db.Exec("UPDATE action_parameter SET type = 'ssh-key', value = $1 WHERE id = $2", newValue, param.ID); err != nil {
			log.Error("cannot update action_parameter %d %s : %v", param.ID, param.Name, err)
			globalError = sdk.WrapError(err, "%v", globalError)
		}
	}

	params, err = getprivateKeyActionParametersEdge(db)
	if err != nil {
		return sdk.WrapError(err, "cannot get private key action parameters edge")
	}

	for _, param := range params {
		newValue := getNewPrivateKeyValue(param.Value)
		if param.Value == "" {
			if _, err := db.Exec("UPDATE action_edge_parameter SET type = 'ssh-key' WHERE id = $1", param.ID); err != nil {
				log.Error("cannot update action_edge_parameter type %d %s : %v", param.ID, param.Name, err)
				globalError = sdk.WrapError(err, "%v", globalError)
			}
			continue
		}
		if newValue == param.Value {
			log.Warning("bad key, cannot get new private key value for param edge %d %s --> Value = %s", param.ID, param.Name, param.Value)
			continue
		}
		if _, err := db.Exec("UPDATE action_edge_parameter SET type = 'ssh-key', value = $1 WHERE id = $2", newValue, param.ID); err != nil {
			log.Error("cannot update action_edge_parameter %d %s : %v", param.ID, param.Name, err)
			globalError = sdk.WrapError(err, "%v", globalError)
		}
	}

	return globalError
}

func getNewPrivateKeyValue(privateKeyValue string) string {
	newValue := privateKeyValue
	switch {
	case strings.HasPrefix(privateKeyValue, "{{.cds.proj."):
		regx := regexp.MustCompile(`{{\.cds\.proj\.(.+)}}`)
		subMatch := regx.FindAllStringSubmatch(privateKeyValue, -1)
		if len(subMatch) > 0 && len(subMatch[0]) > 1 {
			kname := "proj-" + subMatch[0][1]
			newValue = kname
		}
	case strings.HasPrefix(privateKeyValue, "{{.cds.env."):
		regx := regexp.MustCompile(`{{\.cds\.env\.(.+)}}`)
		subMatch := regx.FindAllStringSubmatch(privateKeyValue, -1)
		if len(subMatch) > 0 && len(subMatch[0]) > 1 {
			kname := "env-" + subMatch[0][1]
			newValue = kname
		}
	case strings.HasPrefix(privateKeyValue, "{{.cds.app."):
		regx := regexp.MustCompile(`{{\.cds\.app\.(.+)}}`)
		subMatch := regx.FindAllStringSubmatch(privateKeyValue, -1)
		if len(subMatch) > 0 && len(subMatch[0]) > 1 {
			kname := "app-" + subMatch[0][1]
			newValue = kname
		}
	}

	return newValue
}

func getprivateKeyActionParametersEdge(db *gorp.DbMap) ([]sdk.Parameter, error) {
	query := "SELECT id, name, value, description, advanced FROM action_edge_parameter WHERE name = 'privateKey' AND advanced <> true AND description LIKE '%Set the private key to be able to git clone from ssh%'"
	rows, err := db.Query(query)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get parameters")
	}
	defer rows.Close()

	var params []sdk.Parameter
	for rows.Next() {
		var parameter sdk.Parameter
		if err := rows.Scan(&parameter.ID, &parameter.Name, &parameter.Value, &parameter.Description, &parameter.Advanced); err != nil {
			log.Error("cannot scan parameter : %v", err)
			continue
		}
		params = append(params, parameter)
	}
	return params, nil
}

func getprivateKeyActionParameters(db *gorp.DbMap) ([]sdk.Parameter, error) {
	query := "SELECT id, name, value, description, advanced FROM action_parameter WHERE name = 'privateKey' AND advanced <> true AND description LIKE '%Set the private key to be able to git clone from ssh%'"
	rows, err := db.Query(query)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get parameters")
	}
	defer rows.Close()

	var params []sdk.Parameter
	for rows.Next() {
		var parameter sdk.Parameter
		if err := rows.Scan(&parameter.ID, &parameter.Name, &parameter.Value, &parameter.Description, &parameter.Advanced); err != nil {
			log.Error("cannot scan parameter : %v", err)
			continue
		}
		params = append(params, parameter)
	}
	return params, nil
}
