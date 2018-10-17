package application

import (
	"encoding/base64"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
)

// EncryptVCSStrategyPassword Encrypt vcs password
func EncryptVCSStrategyPassword(app *sdk.Application) error {
	encryptedPwd, err := secret.Encrypt([]byte(app.RepositoryStrategy.Password))
	if err != nil {
		return sdk.WrapError(err, "Unable to encrypt password")
	}

	app.RepositoryStrategy.Password = base64.StdEncoding.EncodeToString(encryptedPwd)
	return nil
}

// DecryptVCSStrategyPassword Decrypt vs password
func DecryptVCSStrategyPassword(app *sdk.Application) error {
	if app.RepositoryStrategy.Password == "" {
		return nil
	}
	encryptedPassword, err64 := base64.StdEncoding.DecodeString(app.RepositoryStrategy.Password)
	if err64 != nil {
		return sdk.WrapError(err64, "EncryptVCSStrategyPassword> Unable to decoding password")
	}

	clearPWD, err := secret.Decrypt([]byte(encryptedPassword))
	if err != nil {
		return sdk.WrapError(err, "Unable to decrypt password")
	}

	app.RepositoryStrategy.Password = string(clearPWD)
	return nil
}

// CountApplicationByVcsConfigurationKeys counts key use in application vcs configuration for the given project
func CountApplicationByVcsConfigurationKeys(db gorp.SqlExecutor, projectKey string, vcsName string) ([]string, error) {
	query := `
		SELECT prequery.name FROM 
		(
			SELECT application.name, vcs_strategy->>'ssh_key' as sshkey, vcs_strategy->>'pgp_key' as pgpkey from application
			JOIN project on application.project_id = project.id
			WHERE project.projectkey = $1
		) prequery
		WHERE sshkey = $2 OR pgpkey = $2`
	var appsName []string
	if _, err := db.Select(&appsName, query, projectKey, vcsName); err != nil {
		return nil, sdk.WrapError(err, "Cannot count keyName in vcs configuration")
	}
	return appsName, nil
}

// GetNameByVCSServer Get the name of application that are linked to the given repository manager
func GetNameByVCSServer(db gorp.SqlExecutor, vcsName string, projectKey string) ([]string, error) {
	var appsName []string
	query := `
		SELECT application.name
		FROM application
		JOIN project on project.id = application.project_id
		WHERE project.projectkey = $1 AND application.vcs_server = $2
	`
	if _, err := db.Select(&appsName, query, projectKey, vcsName); err != nil {
		return nil, sdk.WrapError(err, "Unable to list application name")
	}
	return appsName, nil
}
