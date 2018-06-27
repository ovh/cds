package action

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// CreateBuiltinActions add builtin actions in database if needed
func CreateBuiltinActions(db *gorp.DbMap) error {
	// ----------------------------------- Script ---------------------------
	script := sdk.NewAction(sdk.ScriptAction)
	script.Type = sdk.BuiltinAction
	script.Description = `CDS Builtin Action.
Execute a script, written in script attribute.`
	script.Parameter(sdk.Parameter{
		Name: "script",
		Description: `Content of your script.
You can put #!/bin/bash, or #!/bin/perl at first line.
Make sure that the binary used is in
the pre-requisites of action`,
		Type: sdk.TextParameter})
	if err := checkBuiltinAction(db, script); err != nil {
		return err
	}

	// ----------------------------------- JUnit    ---------------------------
	junit := sdk.NewAction(sdk.JUnitAction)
	junit.Type = sdk.BuiltinAction
	junit.Description = `CDS Builtin Action.
Parse given file to extract Unit Test results.`
	junit.Parameter(sdk.Parameter{
		Name:        "path",
		Description: `Path to junit xml file.`,
		Type:        sdk.TextParameter})
	if err := checkBuiltinAction(db, junit); err != nil {
		return err
	}

	// ----------------------------------- Git clone    -----------------------
	// TODO: WRITE SQL TO UPDATE ACTION
	gitclone := sdk.NewAction(sdk.GitCloneAction)
	gitclone.Type = sdk.BuiltinAction
	gitclone.Description = `CDS Builtin Action.
Clone a repository into a new directory.`

	gitclone.Parameter(sdk.Parameter{
		Name: "url",
		Description: `URL must contain information about the transport protocol, the address of the remote server, and the path to the repository.
If your application is linked to a repository, you can use {{.git.url}} (clone over ssh) or {{.git.http_url}} (clone over https)`,
		Value: "{{.git.url}}",
		Type:  sdk.StringParameter,
	})
	gitclone.Parameter(sdk.Parameter{
		Name:  "privateKey",
		Value: "",
		Description: `Set the private key to be able to git clone from ssh.
You can create an application key named 'app-key' and use it in this action.
The public key have to be granted on your repository`,
		Type: sdk.KeySSHParameter,
	})
	gitclone.Parameter(sdk.Parameter{
		Name:        "user",
		Description: "Set the user to be able to git clone from https with authentication",
		Type:        sdk.StringParameter,
	})
	gitclone.Parameter(sdk.Parameter{
		Name:        "password",
		Description: "Set the password to be able to git clone from https with authentication",
		Type:        sdk.StringParameter,
	})
	gitclone.Parameter(sdk.Parameter{
		Name:        "branch",
		Description: "Instead of pointing the newly created HEAD to the branch pointed to by the cloned repository’s HEAD, point to {{.git.branch}} branch instead.",
		Value:       "{{.git.branch}}",
		Type:        sdk.StringParameter,
	})
	gitclone.Parameter(sdk.Parameter{
		Name:        "commit",
		Description: "Set the current branch head (HEAD) to the commit.",
		Value:       "{{.git.hash}}",
		Type:        sdk.StringParameter,
	})
	gitclone.Parameter(sdk.Parameter{
		Name:        "directory",
		Description: "The name of a directory to clone into.",
		Value:       "{{.cds.workspace}}",
		Type:        sdk.StringParameter,
	})
	gitclone.Requirement("git", sdk.BinaryRequirement, "git")

	if err := checkBuiltinAction(db, gitclone); err != nil {
		return err
	}

	// ----------------------------------- Checkout Application    -----------------------
	checkoutApplication := sdk.NewAction(sdk.CheckoutApplicationAction)
	checkoutApplication.Type = sdk.BuiltinAction
	checkoutApplication.Description = `CDS Builtin Action.
Checkout a repository into a new directory.`

	checkoutApplication.Parameter(sdk.Parameter{
		Name:        "directory",
		Description: "The name of a directory to clone into.",
		Value:       "{{.cds.workspace}}",
		Type:        sdk.StringParameter,
	})
	checkoutApplication.Requirement("git", sdk.BinaryRequirement, "git")

	if err := checkBuiltinAction(db, checkoutApplication); err != nil {
		return err
	}

	// ----------------------------------- Deploy Application    -----------------------
	deployApplication := sdk.NewAction(sdk.DeployApplicationAction)
	deployApplication.Type = sdk.BuiltinAction
	deployApplication.Description = `CDS Builtin Action.
Deploy an application of a platform.`

	if err := checkBuiltinAction(db, deployApplication); err != nil {
		return err
	}

	// ----------------------------------- Git tag    -----------------------
	gittag := sdk.NewAction(sdk.GitTagAction)
	gittag.Type = sdk.BuiltinAction
	gittag.Description = `CDS Builtin Action.
Tag the current branch and push it.
Semver used if fully compatible with https://semver.org/
`

	gittag.Parameter(sdk.Parameter{
		Name:        "tagPrerelease",
		Description: "Prerelase version of the tag. Example: alpha on a tag 1.0.0 will return 1.0.0-apha",
		Value:       "",
		Type:        sdk.StringParameter,
	})
	gittag.Parameter(sdk.Parameter{
		Name:        "tagLevel",
		Description: "Set the level of the tag. Must be 'major' or 'minor' or 'patch'",
		Value:       "",
		Type:        sdk.StringParameter,
	})
	gittag.Parameter(sdk.Parameter{
		Name:        "tagMetadata",
		Description: "Metadata of the tag. Example: cds.42 on a tag 1.0.0 will return 1.0.0+cds.42",
		Value:       "",
		Type:        sdk.StringParameter,
	})
	gittag.Parameter(sdk.Parameter{
		Name:        "tagMessage",
		Description: "Set a message for the tag.",
		Value:       "",
		Type:        sdk.StringParameter,
	})
	gittag.Parameter(sdk.Parameter{
		Name:        "path",
		Description: "The path to your git directory.",
		Value:       "{{.cds.workspace}}",
		Type:        sdk.StringParameter,
	})
	gittag.Requirement("git", sdk.BinaryRequirement, "git")
	gittag.Requirement("gpg", sdk.BinaryRequirement, "gpg")

	if err := checkBuiltinAction(db, gittag); err != nil {
		return err
	}

	// ----------------------------------- Git Release -----------------------
	gitrelease := sdk.NewAction(sdk.ReleaseAction)
	gitrelease.Type = sdk.BuiltinAction
	gitrelease.Description = `CDS Builtin Action. Make a release using repository manager.`

	gitrelease.Parameter(sdk.Parameter{
		Name:        "tag",
		Description: "Tag name.",
		Value:       "{{.cds.release.version}}",
		Type:        sdk.StringParameter,
	})
	gitrelease.Parameter(sdk.Parameter{
		Name:        "title",
		Value:       "",
		Description: "Set a title for the release",
		Type:        sdk.StringParameter,
	})
	gitrelease.Parameter(sdk.Parameter{
		Name:        "releaseNote",
		Description: "Set a release note for the release",
		Type:        sdk.TextParameter,
	})
	gitrelease.Parameter(sdk.Parameter{
		Name:        "artifacts",
		Description: "Set a list of artifacts, separate by , . You can also use regexp.",
		Type:        sdk.StringParameter,
	})
	return checkBuiltinAction(db, gitrelease)
}

// checkBuiltinAction add builtin actions in database if needed
func checkBuiltinAction(db *gorp.DbMap, a *sdk.Action) error {
	var name string
	err := db.QueryRow(`SELECT action.name FROM action WHERE action.name = $1 and action.type = $2`, a.Name, sdk.BuiltinAction).Scan(&name)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if err != nil && err == sql.ErrNoRows {
		if errcreate := createBuiltinAction(db, a); errcreate != nil {
			return errcreate
		}
	}

	return nil
}

func createBuiltinAction(db *gorp.DbMap, a *sdk.Action) error {
	tx, errb := db.Begin()
	if errb != nil {
		return errb
	}
	defer tx.Rollback()

	log.Info("createBuiltinAction> create builtin action %s", a.Name)
	if err := InsertAction(tx, a, true); err != nil {
		return err
	}

	return tx.Commit()
}

// CreateBuiltinArtifactActions  Create Action BuiltinArtifact
func CreateBuiltinArtifactActions(db *gorp.DbMap) error {
	var name string
	query := `SELECT action.name FROM action where action.name = $1 and action.type = $2`

	// Check ArtifactUpload action
	err := db.QueryRow(query, sdk.ArtifactUpload, sdk.BuiltinAction).Scan(&name)
	if err != nil && err == sql.ErrNoRows {
		err = createBuiltinArtifactUploadAction(db)
		if err != nil {
			return sdk.WrapError(err, "CreateBuiltinArtifactActions> cannot create builtin artifact upload action")
		}
	}

	// Check ArtifactDownload action
	err = db.QueryRow(query, sdk.ArtifactDownload, sdk.BuiltinAction).Scan(&name)
	if err != nil && err == sql.ErrNoRows {
		err = createBuiltinArtifactDownloadAction(db)
		if err != nil {
			return sdk.WrapError(err, "CreateBuiltinArtifactActions> cannot create builtin artifact download action")
		}
	}

	return nil
}

func createBuiltinArtifactUploadAction(db *gorp.DbMap) error {
	upload := sdk.NewAction(sdk.ArtifactUpload)
	upload.Type = sdk.BuiltinAction
	upload.Parameter(sdk.Parameter{
		Name:        "path",
		Type:        sdk.StringParameter,
		Description: "Path of file to upload, example: ./src/yourFile.json"})
	upload.Parameter(sdk.Parameter{
		Name:        "tag",
		Type:        sdk.StringParameter,
		Description: "Artifact will be uploaded with a tag, generally {{.cds.version}}",
		Value:       "{{.cds.version}}"})
	upload.Parameter(sdk.Parameter{
		Name:        "enabled",
		Type:        sdk.BooleanParameter,
		Description: "Enable artifact upload",
		Value:       "true"})

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	log.Info("createBuiltinArtifactUploadAction> create builtin action %s", upload.Name)
	if err := InsertAction(tx, upload, true); err != nil {
		return sdk.WrapError(err, "CreateBuiltinArtifactActions> createBuiltinArtifactUploadAction err")
	}

	return tx.Commit()
}

func createBuiltinArtifactDownloadAction(db *gorp.DbMap) error {
	dl := sdk.NewAction(sdk.ArtifactDownload)
	dl.Type = sdk.BuiltinAction
	dl.Parameter(sdk.Parameter{
		Name:        "path",
		Description: "Path where artifacts will be downloaded",
		Type:        sdk.StringParameter})
	dl.Parameter(sdk.Parameter{
		Name:        "tag",
		Description: "Artifact are uploaded with a tag, generally {{.cds.version}}",
		Type:        sdk.StringParameter})
	dl.Parameter(sdk.Parameter{
		Name:        "pipeline",
		Description: "Pipeline from where artifacts will be downloaded, generally {{.cds.pipeline}} or {{.cds.parent.pipeline}}",
		Type:        sdk.StringParameter})
	dl.Parameter(sdk.Parameter{
		Name:        "application",
		Description: "Application from where artifacts will be downloaded, generally {{.cds.application}}",
		Type:        sdk.StringParameter})
	dl.Parameter(sdk.Parameter{
		Name:        "enabled",
		Type:        sdk.BooleanParameter,
		Description: "Enable artifact download",
		Value:       "true"})
	dl.Parameter(sdk.Parameter{
		Name:        "pattern",
		Type:        sdk.StringParameter,
		Description: "Empty: download all files. Otherwise, enter regexp pattern to choose file: (fileA|fileB)",
		Value:       ""})

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	log.Info("createBuiltinArtifactDownloadAction> create builtin action %s", dl.Name)
	if err := InsertAction(tx, dl, true); err != nil {
		return sdk.WrapError(err, "CreateBuiltinArtifactActions> createBuiltinArtifactDownloadAction err")
	}

	return tx.Commit()
}
