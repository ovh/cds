package project

import (
	"database/sql"
	"encoding/base64"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/platform"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
)

// PostGet is a db hook
func (pp *dbProjectPlatform) PostGet(db gorp.SqlExecutor) error {
	model, err := platform.LoadModel(db, pp.PlatformModelID)
	if err != nil {
		return sdk.WrapError(err, "dbProjectPlatform.PostGet> Cannot load model")
	}
	pp.Model = model

	query := "SELECT config FROM project_platform where id = $1"
	s, err := db.SelectNullStr(query, pp.ID)
	if err != nil {
		return sdk.WrapError(err, "dbProjectPlatform.PostGet> Cannot get config")
	}
	if err := gorpmapping.JSONNullString(s, &pp.Config); err != nil {
		return err
	}
	return nil
}

// DeletePlatform
func DeletePlatform(db gorp.SqlExecutor, platform sdk.ProjectPlatform) error {
	pp := dbProjectPlatform(platform)
	if _, err := db.Delete(&pp); err != nil {
		return sdk.WrapError(err, "DeletePlatform> Cannot remove project platform")
	}
	return nil
}

// LoadPlatformsByName Load a platform by project key and its name
func LoadPlatformsByName(db gorp.SqlExecutor, key string, name string, clearPwd bool) (sdk.ProjectPlatform, error) {
	var pp dbProjectPlatform
	query := `
		SELECT project_platform.*
		FROM project_platform
		JOIN project ON project.id = project_platform.project_id
		WHERE project.projectkey = $1 AND project_platform.name = $2
	`
	if err := db.SelectOne(&pp, query, key, name); err != nil {
		return sdk.ProjectPlatform{}, sdk.WrapError(err, "LoadPlatformsByName> Cannot load platform")
	}
	p := sdk.ProjectPlatform(pp)
	for k, v := range p.Config {
		if v.Type == sdk.PlatformConfigTypePassword {
			if clearPwd {
				decryptedValue, errD := decryptPlatformValue(v.Value)
				if errD != nil {
					return p, sdk.WrapError(errD, "LoadPlatformsByName> Cannot decrypt value")
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

// LoadPlatformsByID load project platforms by project id
func LoadPlatformsByID(db gorp.SqlExecutor, id int64, clearPassword bool) ([]sdk.ProjectPlatform, error) {
	platforms := []sdk.ProjectPlatform{}

	var res []dbProjectPlatform
	if _, err := db.Select(&res, "SELECT * from project_platform WHERE project_id = $1", id); err != nil {
		if err == sql.ErrNoRows {
			return platforms, nil
		}
		return nil, err
	}

	platforms = make([]sdk.ProjectPlatform, len(res))
	for i := range res {
		pp := &res[i]
		if err := pp.PostGet(db); err != nil {
			return nil, sdk.WrapError(err, "LoadPlatformByID> Cannot post get")
		}

		for k, v := range pp.Config {
			if v.Type == sdk.PlatformConfigTypePassword {
				if clearPassword {
					secret, errD := decryptPlatformValue(v.Value)
					if errD != nil {
						return nil, sdk.WrapError(errD, "LoadPlatformByID> Cannot decrypt password")
					}
					v.Value = string(secret)
					pp.Config[k] = v
				} else {
					v.Value = sdk.PasswordPlaceholder
					pp.Config[k] = v
				}
			}
		}
		platforms[i] = sdk.ProjectPlatform(*pp)
	}
	return platforms, nil
}

func decryptPlatformValue(v string) (string, error) {
	b, err64 := base64.StdEncoding.DecodeString(v)
	if err64 != nil {
		return "", sdk.WrapError(err64, "decryptPlatformValue> cannot decode string")
	}
	secret, errD := secret.Decrypt(b)
	if errD != nil {
		return "", sdk.WrapError(errD, "decryptPlatformValue> Cannot decrypt password")
	}
	return string(secret), nil
}

func encryptPlatformValue(v string) (string, error) {
	encryptedSecret, errE := secret.Encrypt([]byte(v))
	if errE != nil {
		return "", sdk.WrapError(errE, "encryptPlatformValue> Cannot encrypt password")
	}
	return base64.StdEncoding.EncodeToString(encryptedSecret), nil
}

// InsertPlatform inserts a project platform
func InsertPlatform(db gorp.SqlExecutor, pp *sdk.ProjectPlatform) error {
	for k, v := range pp.Config {
		if v.Type == sdk.PlatformConfigTypePassword {
			s, errS := encryptPlatformValue(v.Value)
			if errS != nil {
				return sdk.WrapError(errS, "InsertPlatform> Cannot encrypt password")
			}
			v.Value = string(s)
			pp.Config[k] = v
		}
	}
	ppDb := dbProjectPlatform(*pp)
	if err := db.Insert(&ppDb); err != nil {
		return sdk.WrapError(err, "InsertPlatform> Cannot insert project platform")
	}
	*pp = sdk.ProjectPlatform(ppDb)
	return nil
}

// UpdatePlatform Update a project platform
func UpdatePlatform(db gorp.SqlExecutor, pp sdk.ProjectPlatform) error {
	for k, v := range pp.Config {
		if v.Type == sdk.PlatformConfigTypePassword {
			s, errS := encryptPlatformValue(v.Value)
			if errS != nil {
				return sdk.WrapError(errS, "UpdatePlatform> Cannot encrypt password")
			}
			v.Value = string(s)
			pp.Config[k] = v
		}
	}
	ppDb := dbProjectPlatform(pp)
	if _, err := db.Update(&ppDb); err != nil {
		return sdk.WrapError(err, "UpdatePlatform> Cannot update project platform")
	}
	return nil
}

// PostUpdate is a db hook
func (pp *dbProjectPlatform) PostUpdate(db gorp.SqlExecutor) error {
	configB, err := gorpmapping.JSONToNullString(pp.Config)
	if err != nil {
		return sdk.WrapError(err, "PostInsert.projectPlatform> Cannot post insert project platform")
	}

	if _, err := db.Exec("UPDATE project_platform set config = $1 WHERE id = $2", configB, pp.ID); err != nil {
		return sdk.WrapError(err, "PostInsert.projectPlatform> Cannot update config")
	}
	return nil
}

// PostInsert is a db hook
func (pp *dbProjectPlatform) PostInsert(db gorp.SqlExecutor) error {
	if err := pp.PostUpdate(db); err != nil {
		return sdk.WrapError(err, "PostInsert.projectPlatform> Cannot update")
	}
	return nil
}
