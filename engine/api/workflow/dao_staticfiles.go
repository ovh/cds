package workflow

import (
	"database/sql"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// DEPRECATED
func loadStaticFilesByNodeRunID(db gorp.SqlExecutor, nodeRunID int64) ([]sdk.StaticFiles, error) {
	var dbstaticFiles []dbStaticFiles
	if _, err := db.Select(&dbstaticFiles, `SELECT
			id,
			name,
			entrypoint,
			created,
			public_url,
			workflow_node_run_id
		FROM workflow_node_run_static_files WHERE workflow_node_run_id = $1`, nodeRunID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	staticFiles := make([]sdk.StaticFiles, len(dbstaticFiles))
	for i := range dbstaticFiles {
		staticFiles[i] = sdk.StaticFiles(dbstaticFiles[i])
	}
	return staticFiles, nil
}

// DEPRECATED
// InsertStaticFiles insert in table workflow_artifacts
func InsertStaticFiles(db gorp.SqlExecutor, sf *sdk.StaticFiles) error {
	sf.Created = time.Now()
	dbstaticFiles := dbStaticFiles(*sf)
	if err := db.Insert(&dbstaticFiles); err != nil {
		return err
	}
	sf.ID = dbstaticFiles.ID
	return nil
}
