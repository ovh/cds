package action

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
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

	// ----------------------------------- Notif  ---------------------------
	notif := sdk.NewAction(sdk.NotifAction)
	notif.Description = `CDS Builtin Action. This action can be used to send
information to notification systems.
Each notification system can interpret this
notification as desired.

You can write content in a file, using messagefile
attribute.

Consult documentation for more information`
	notif.Type = sdk.BuiltinAction
	notif.Parameter(sdk.Parameter{
		Name:  "destination",
		Value: "all",
		Description: `Destination of notification: email, tat, slack, jabber...
Check CDS Documentation for available type. Default to all`,
		Type: sdk.StringParameter})
	notif.Parameter(sdk.Parameter{
		Name:        "title",
		Description: "Title of notification",
		Type:        sdk.StringParameter})
	notif.Parameter(sdk.Parameter{
		Name:        "message",
		Description: "Message of notification (optional)",
		Type:        sdk.TextParameter})
	notif.Parameter(sdk.Parameter{
		Name:        "messagefile",
		Value:       "",
		Description: "Message could be in this file (optional)",
		Type:        sdk.StringParameter})

	if err := checkBuiltinAction(db, notif); err != nil {
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

	return nil
}

// checkBuiltinAction add builtin actions in database if needed
func checkBuiltinAction(db *gorp.DbMap, a *sdk.Action) error {
	var name string
	err := db.QueryRow(`SELECT action.name FROM action WHERE action.name = $1`, a.Name).Scan(&name)
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

	if err := InsertAction(tx, a, true); err != nil {
		return err
	}

	return tx.Commit()
}

// CreateBuiltinArtifactActions  Create Action BuiltinArtifact
func CreateBuiltinArtifactActions(db *gorp.DbMap) error {
	var name string
	query := `SELECT action.name FROM action where action.name = $1`

	// Check ArtifactUpload action
	err := db.QueryRow(query, sdk.ArtifactUpload).Scan(&name)
	if err != nil && err == sql.ErrNoRows {
		err = createBuiltinArtifactUploadAction(db)
		if err != nil {
			log.Warning("CreateBuiltinArtifactActions> CreateBuiltinArtifactActions err:%s", err.Error())
			return err
		}
	}

	// Check ArtifactDownload action
	err = db.QueryRow(query, sdk.ArtifactDownload).Scan(&name)
	if err != nil && err == sql.ErrNoRows {
		err = createBuiltinArtifactDownloadAction(db)
		if err != nil {
			log.Warning("CreateBuiltinArtifactActions> createBuiltinArtifactDownloadAction err:%s", err.Error())
			return err
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

	err = InsertAction(tx, upload, true)
	if err != nil {
		log.Warning("CreateBuiltinArtifactActions> createBuiltinArtifactUploadAction err:%s", err.Error())
		return err
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

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = InsertAction(tx, dl, true)
	if err != nil {
		log.Warning("CreateBuiltinArtifactActions> createBuiltinArtifactDownloadAction err:%s", err.Error())
		return err
	}

	return tx.Commit()
}
