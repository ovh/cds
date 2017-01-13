package artifact

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"

)

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

	err = action.InsertAction(tx, upload, true)
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

	err = action.InsertAction(tx, dl, true)
	if err != nil {
		log.Warning("CreateBuiltinArtifactActions> createBuiltinArtifactDownloadAction err:%s", err.Error())
		return err
	}

	return tx.Commit()
}
