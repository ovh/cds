package integration

import (
	"database/sql"
	"encoding/base64"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
)

// PostGet is a db hook
func (pp *dbProjectIntegration) PostGet(db gorp.SqlExecutor) error {
	model, err := LoadModel(db, pp.IntegrationModelID, false)
	if err != nil {
		return sdk.WrapError(err, "Cannot load model")
	}
	pp.Model = model

	query := "SELECT config FROM project_integration where id = $1"
	s, err := db.SelectNullStr(query, pp.ID)
	if err != nil {
		return sdk.WrapError(err, "Cannot get config")
	}
	if err := gorpmapping.JSONNullString(s, &pp.Config); err != nil {
		return err
	}
	return nil
}

// DeleteIntegration deletes a integration
func DeleteIntegration(db gorp.SqlExecutor, integration sdk.ProjectIntegration) error {
	pp := dbProjectIntegration(integration)
	if _, err := db.Delete(&pp); err != nil {
		return sdk.WrapError(err, "Cannot remove integration")
	}
	return nil
}

// LoadProjectIntegrationByName Load a integration by project key and its name
func LoadProjectIntegrationByName(db gorp.SqlExecutor, key string, name string, clearPwd bool) (sdk.ProjectIntegration, error) {
	var pp dbProjectIntegration
	query := `
		SELECT project_integration.*
		FROM project_integration
		JOIN project ON project.id = project_integration.project_id
		WHERE project.projectkey = $1 AND project_integration.name = $2
	`
	if err := db.SelectOne(&pp, query, key, name); err != nil {
		return sdk.ProjectIntegration{}, sdk.WithStack(err)
	}
	p := sdk.ProjectIntegration(pp)
	for k, v := range p.Config {
		if v.Type == sdk.IntegrationConfigTypePassword {
			if clearPwd {
				decryptedValue, errD := decryptIntegrationValue(v.Value)
				if errD != nil {
					return p, sdk.WrapError(errD, "LoadProjectIntegrationByName> Cannot decrypt value")
				}
				v.Value = decryptedValue
				p.Config[k] = v
			} else {
				v.Value = sdk.PasswordPlaceholder
				p.Config[k] = v
			}
		}
	}

	return p, nil
}

// LoadProjectIntegrationByID returns integration, selecting by its id
func LoadProjectIntegrationByID(db gorp.SqlExecutor, id int64, clearPassword bool) (*sdk.ProjectIntegration, error) {
	var pp dbProjectIntegration
	if err := db.SelectOne(&pp, "SELECT * from project_integration WHERE id = $1", id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	for k, v := range pp.Config {
		if v.Type == sdk.IntegrationConfigTypePassword {
			if clearPassword {
				secret, errD := decryptIntegrationValue(v.Value)
				if errD != nil {
					return nil, sdk.WrapError(errD, "LoadIntegrationByID> Cannot decrypt password")
				}
				v.Value = string(secret)
				pp.Config[k] = v
			} else {
				v.Value = sdk.PasswordPlaceholder
				pp.Config[k] = v
			}
		}
	}
	res := sdk.ProjectIntegration(pp)
	return &res, nil
}

// LoadIntegrationsByProjectID load integration integrations by project id
func LoadIntegrationsByProjectID(db gorp.SqlExecutor, id int64, clearPassword bool) ([]sdk.ProjectIntegration, error) {
	integrations := []sdk.ProjectIntegration{}

	var res []dbProjectIntegration
	if _, err := db.Select(&res, "SELECT * from project_integration WHERE project_id = $1", id); err != nil {
		if err == sql.ErrNoRows {
			return integrations, nil
		}
		return nil, err
	}

	integrations = make([]sdk.ProjectIntegration, len(res))
	for i := range res {
		pp := &res[i]
		if err := pp.PostGet(db); err != nil {
			return nil, sdk.WrapError(err, "Cannot post get")
		}

		for k, v := range pp.Config {
			if v.Type == sdk.IntegrationConfigTypePassword {
				if clearPassword {
					secret, errD := decryptIntegrationValue(v.Value)
					if errD != nil {
						return nil, sdk.WrapError(errD, "LoadIntegrationByID> Cannot decrypt password")
					}
					v.Value = string(secret)
					pp.Config[k] = v
				} else {
					v.Value = sdk.PasswordPlaceholder
					pp.Config[k] = v
				}
			}
		}
		integrations[i] = sdk.ProjectIntegration(*pp)
	}
	return integrations, nil
}

func decryptIntegrationValue(v string) (string, error) {
	b, err64 := base64.StdEncoding.DecodeString(v)
	if err64 != nil {
		return "", sdk.WrapError(err64, "decryptIntegrationValue> cannot decode string")
	}
	secret, errD := secret.Decrypt(b)
	if errD != nil {
		return "", sdk.WrapError(errD, "decryptIntegrationValue> Cannot decrypt password")
	}
	return string(secret), nil
}

func encryptIntegrationValue(v string) (string, error) {
	encryptedSecret, errE := secret.Encrypt([]byte(v))
	if errE != nil {
		return "", sdk.WrapError(errE, "encryptIntegrationValue> Cannot encrypt password")
	}
	return base64.StdEncoding.EncodeToString(encryptedSecret), nil
}

// InsertIntegration inserts a integration
func InsertIntegration(db gorp.SqlExecutor, pp *sdk.ProjectIntegration) error {
	for k, v := range pp.Config {
		if v.Type == sdk.IntegrationConfigTypePassword {
			s, errS := encryptIntegrationValue(v.Value)
			if errS != nil {
				return sdk.WrapError(errS, "InsertIntegration> Cannot encrypt password")
			}
			v.Value = string(s)
			pp.Config[k] = v
		}
	}
	ppDb := dbProjectIntegration(*pp)
	if err := db.Insert(&ppDb); err != nil {
		return sdk.WrapError(err, "Cannot insert integration")
	}
	*pp = sdk.ProjectIntegration(ppDb)
	return nil
}

// UpdateIntegration Update a integration
func UpdateIntegration(db gorp.SqlExecutor, pp sdk.ProjectIntegration) error {
	for k, v := range pp.Config {
		if v.Type == sdk.IntegrationConfigTypePassword {
			s, errS := encryptIntegrationValue(v.Value)
			if errS != nil {
				return sdk.WrapError(errS, "UpdateIntegration> Cannot encrypt password")
			}
			v.Value = string(s)
			pp.Config[k] = v
		}
	}
	ppDb := dbProjectIntegration(pp)
	if _, err := db.Update(&ppDb); err != nil {
		return sdk.WrapError(err, "Cannot update integration")
	}
	return nil
}

// PostUpdate is a db hook
func (pp *dbProjectIntegration) PostUpdate(db gorp.SqlExecutor) error {
	configB, err := gorpmapping.JSONToNullString(pp.Config)
	if err != nil {
		return sdk.WrapError(err, "Cannot post insert integration")
	}

	if _, err := db.Exec("UPDATE project_integration set config = $1 WHERE id = $2", configB, pp.ID); err != nil {
		return sdk.WrapError(err, "Cannot update config")
	}
	return nil
}

// PostInsert is a db hook
func (pp *dbProjectIntegration) PostInsert(db gorp.SqlExecutor) error {
	if err := pp.PostUpdate(db); err != nil {
		return sdk.WrapError(err, "Cannot update")
	}
	return nil
}
