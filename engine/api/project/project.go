package project

import (
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/sdk"
)

// InsertProject insert given project into given database
func InsertProject(db gorp.SqlExecutor, p *sdk.Project) error {
	if p.Name == "" {
		return sdk.ErrInvalidName
	}
	query := `INSERT INTO project (projectKey, name) VALUES($1,$2) RETURNING id`
	err := db.QueryRow(query, p.Key, p.Name).Scan(&p.ID)
	return err
}

// UpdateProjectDB set new project name in database
func UpdateProjectDB(db gorp.SqlExecutor, projectKey, projectName string) (time.Time, error) {
	var lastModified time.Time
	query := `UPDATE project SET name=$1, last_modified=current_timestamp WHERE projectKey=$2 RETURNING last_modified`
	err := db.QueryRow(query, projectName, projectKey).Scan(&lastModified)
	return lastModified, err
}

// AddKeyPairToProject generate a ssh key pair and add them as project variables
func AddKeyPairToProject(db gorp.SqlExecutor, proj *sdk.Project, keyname string) error {

	pub, priv, errGenerate := keys.Generatekeypair(keyname)
	if errGenerate != nil {
		return errGenerate
	}

	v := sdk.Variable{
		Name:  keyname,
		Type:  sdk.KeyVariable,
		Value: priv,
	}

	if err := InsertVariableInProject(db, proj, v); err != nil {
		return err
	}

	p := sdk.Variable{
		Name:  keyname + ".pub",
		Type:  sdk.TextVariable,
		Value: pub,
	}

	return InsertVariableInProject(db, proj, p)
}
