package artifact

import (
	"database/sql"
	"io"
	"strings"
	"sync"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// LoadArtifactByHash retrieves an artiface using its download hash
func LoadArtifactByHash(db gorp.SqlExecutor, hash string) (*sdk.Artifact, error) {
	art := &sdk.Artifact{}
	query := `SELECT artifact.id, artifact.name, artifact.tag, 
		  pipeline.name, project.projectKey, application.name, environment.name,
		  artifact.size, artifact.perm, artifact.md5sum, artifact.object_path
		  FROM artifact
		  JOIN pipeline ON artifact.pipeline_id = pipeline.id
		  JOIN project ON pipeline.project_id = project.id
		  JOIN application ON application.id = artifact.application_id
		  JOIN environment ON environment.id = artifact.environment_id
		  WHERE download_hash = $1`

	var md5sum, objectpath sql.NullString
	var size, perm sql.NullInt64
	err := db.QueryRow(query, hash).Scan(&art.ID, &art.Name, &art.Tag, &art.Pipeline, &art.Project, &art.Application, &art.Environment, &size, &perm, &md5sum, &objectpath)
	if err != nil {
		return nil, err
	}
	if md5sum.Valid {
		art.MD5sum = md5sum.String
	}
	if objectpath.Valid {
		art.ObjectPath = objectpath.String
	}
	if size.Valid {
		art.Size = size.Int64
	}
	if perm.Valid {
		art.Perm = uint32(perm.Int64)
	}
	return art, nil
}

// LoadArtifactsByBuildNumber Load artifact by pipeline ID and buildNUmber
func LoadArtifactsByBuildNumber(db gorp.SqlExecutor, pipelineID int64, applicationID int64, buildNumber int64, environmentID int64) ([]sdk.Artifact, error) {
	query := `SELECT 	artifact.id, 
						artifact.name, 
						artifact.tag, 
						artifact.download_hash, 
						artifact.size, 
						artifact.perm, 
						artifact.md5sum, 
						artifact.object_path,  
						pipeline.name, 
						project.projectKey, 
						application.name, 
						environment.name
	          FROM artifact
			  JOIN pipeline ON artifact.pipeline_id = pipeline.id
			  JOIN project ON pipeline.project_id = project.id
			  JOIN application ON application.id = artifact.application_id
			  JOIN environment ON environment.id = artifact.environment_id		   
			  WHERE build_number = $1 
			  AND pipeline_id = $2 
			  AND application_id = $3 
			  AND environment_id = $4
	          ORDER BY artifact.name`

	rows, err := db.Query(query, buildNumber, pipelineID, applicationID, environmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	arts := []sdk.Artifact{}
	for rows.Next() {
		art := sdk.Artifact{}
		var md5sum, objectpath sql.NullString
		var size, perm sql.NullInt64
		err = rows.Scan(&art.ID, &art.Name, &art.Tag, &art.DownloadHash, &size, &perm, &md5sum, &objectpath, &art.Pipeline, &art.Project, &art.Application, &art.Environment)
		if err != nil {
			return nil, err
		}
		if md5sum.Valid {
			art.MD5sum = md5sum.String
		}
		if objectpath.Valid {
			art.ObjectPath = objectpath.String
		}
		if size.Valid {
			art.Size = size.Int64
		}
		if perm.Valid {
			art.Perm = uint32(perm.Int64)
		}
		arts = append(arts, art)
	}

	getFetchURL(arts)

	log.Info("Artifacts %+v", arts)

	return arts, nil
}

func getFetchURL(arts []sdk.Artifact) {
	//When listing artifacts, try to get a temp url
	if objectstore.Instance().TemporaryURLSupported {
		if store, ok := objectstore.Storage().(objectstore.DriverWithRedirect); ok {
			wg := &sync.WaitGroup{}
			wg.Add(len(arts))
			for i := range arts {
				go func(a *sdk.Artifact) {
					defer wg.Done()
					url, key, err := store.FetchURL(a)
					if err != nil {
						log.Error("artifact>Unable to get Temp URL for %s/%s: %v", a.GetPath(), a.GetName(), err)
						return
					}
					a.TempURL = url
					a.TempURLSecretKey = key
				}(&arts[i])
			}
			wg.Wait()
		}
	}
}

// LoadArtifacts Load artifact by pipeline ID
func LoadArtifacts(db gorp.SqlExecutor, pipelineID int64, applicationID int64, environmentID int64, tag string) ([]sdk.Artifact, error) {
	query := `SELECT 	artifact.id, 
						artifact.name, 
						artifact.tag, 
						artifact.download_hash, 
						artifact.size, 
						artifact.perm, 
						artifact.md5sum, 
						artifact.object_path,  
						pipeline.name, 
						project.projectKey, 
						application.name, 
						environment.name
		FROM artifact
		JOIN pipeline ON artifact.pipeline_id = pipeline.id
		JOIN project ON pipeline.project_id = project.id
		JOIN application ON application.id = artifact.application_id
		JOIN environment ON environment.id = artifact.environment_id	
		WHERE tag = $1 
		AND pipeline_id = $2 
		AND application_id = $3 
		AND environment_id = $4`
	rows, err := db.Query(query, tag, pipelineID, applicationID, environmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var arts []sdk.Artifact
	for rows.Next() {
		art := sdk.Artifact{}
		var md5sum, objectpath sql.NullString
		var size, perm sql.NullInt64
		err = rows.Scan(&art.ID, &art.Name, &art.Tag, &art.DownloadHash, &size, &perm, &md5sum, &objectpath, &art.Pipeline, &art.Project, &art.Application, &art.Environment)
		if err != nil {
			return nil, err
		}
		if md5sum.Valid {
			art.MD5sum = md5sum.String
		}
		if objectpath.Valid {
			art.ObjectPath = objectpath.String
		}
		if size.Valid {
			art.Size = size.Int64
		}
		if perm.Valid {
			art.Perm = uint32(perm.Int64)
		}
		arts = append(arts, art)
	}

	getFetchURL(arts)

	return arts, nil
}

// LoadArtifact Load artifact by ID
func LoadArtifact(db gorp.SqlExecutor, id int64) (*sdk.Artifact, error) {
	query := `SELECT 
			artifact.name, artifact.tag, artifact.download_hash, artifact.size, artifact.perm, artifact.md5sum, artifact.object_path, 
			pipeline.name, project.projectKey, application.name, environment.name FROM artifact
			JOIN pipeline ON artifact.pipeline_id = pipeline.id
			JOIN project ON pipeline.project_id = project.id
			JOIN application ON application.id = artifact.application_id
			JOIN environment ON environment.id = artifact.environment_id
			WHERE artifact.id = $1`

	s := &sdk.Artifact{}
	var md5sum, objectpath sql.NullString
	var size, perm sql.NullInt64
	err := db.QueryRow(query, id).Scan(&s.Name, &s.Tag, &s.DownloadHash, &size, &perm, &md5sum, &objectpath,
		&s.Pipeline, &s.Project, &s.Application, &s.Environment)
	if md5sum.Valid {
		s.MD5sum = md5sum.String
	}
	if objectpath.Valid {
		s.ObjectPath = objectpath.String
	}
	if size.Valid {
		s.Size = size.Int64
	}
	if perm.Valid {
		s.Perm = uint32(perm.Int64)
	}
	return s, err
}

// DeleteArtifactsByApplicationID Delete all artifact related to given application
func DeleteArtifactsByApplicationID(db gorp.SqlExecutor, id int64) error {
	query := `SELECT artifact.name, artifact.tag, pipeline.name, project.projectKey, application.name, environment.name FROM artifact
						JOIN pipeline ON artifact.pipeline_id = pipeline.id
						JOIN project ON pipeline.project_id = project.id
						JOIN application ON application.id = artifact.application_id
						JOIN environment ON environment.id = artifact.environment_id
						WHERE artifact.application_id = $1 FOR UPDATE`

	arts := []sdk.Artifact{}
	rows, errR := db.Query(query, id)
	if errR != nil {
		return sdk.WrapError(errR, "DeleteArtifactsByApplicationID> Cannot select artifact")
	}
	for rows.Next() {
		s := sdk.Artifact{}
		if err := rows.Scan(&s.Name, &s.Tag, &s.Pipeline, &s.Project, &s.Application, &s.Environment); err != nil {
			rows.Close()
			return sdk.WrapError(err, "DeleteArtifactsByApplicationID> Cannot select artifact")
		}
		arts = append(arts, s)
	}
	rows.Close()

	for _, a := range arts {
		if err := objectstore.DeleteArtifact(&a); err != nil && !strings.Contains(err.Error(), "404") {
			return sdk.WrapError(err, "DeleteArtifact> Cannot delete artifact in store")
		}
		query = `DELETE FROM artifact WHERE id = $1`
		if _, err := db.Exec(query, id); err != nil {
			return sdk.WrapError(err, "DeleteArtifact> Cannot delete artifact in DB")
		}
	}
	return nil
}

// DeleteArtifact lock the artifact in database,
// then remove the actual object using storage driver,
// finally remove artifact from database if actual delete is performed
func DeleteArtifact(db gorp.SqlExecutor, id int64) error {
	query := `SELECT artifact.name, artifact.tag, pipeline.name, project.projectKey, application.name, environment.name FROM artifact
						JOIN pipeline ON artifact.pipeline_id = pipeline.id
						JOIN project ON pipeline.project_id = project.id
						JOIN application ON application.id = artifact.application_id
						JOIN environment ON environment.id = artifact.environment_id
						WHERE artifact.id = $1 FOR UPDATE`

	s := sdk.Artifact{}
	if err := db.QueryRow(query, id).Scan(&s.Name, &s.Tag, &s.Pipeline, &s.Project, &s.Application, &s.Environment); err != nil {
		return sdk.WrapError(err, "DeleteArtifact> Cannot select artifact")
	}

	if err := objectstore.DeleteArtifact(&s); err != nil && !strings.Contains(err.Error(), "404") {
		return sdk.WrapError(err, "DeleteArtifact> Cannot delete artifact in store")
	}

	query = `DELETE FROM artifact WHERE id = $1`
	if _, err := db.Exec(query, id); err != nil {
		return sdk.WrapError(err, "DeleteArtifact> Cannot delete artifact in DB")
	}

	return nil
}

func InsertArtifact(db gorp.SqlExecutor, pipelineID, applicationID int64, environmentID int64, art sdk.Artifact) error {
	query := `DELETE FROM "artifact" WHERE name = $1 AND tag = $2 AND pipeline_id = $3 AND application_id = $4 AND environment_id = $5`
	_, err := db.Exec(query, art.Name, art.Tag, pipelineID, applicationID, environmentID)
	if err != nil {
		return err
	}

	query = `INSERT INTO "artifact" 
			(name, tag, pipeline_id, application_id, build_number, environment_id, download_hash, size, perm, md5sum, object_path) 
			VALUES 
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err = db.Exec(query, art.Name, art.Tag, pipelineID, applicationID, art.BuildNumber, environmentID, art.DownloadHash, art.Size, art.Perm, art.MD5sum, art.ObjectPath)
	if err != nil {
		return sdk.WrapError(err, "insertArtifact> Unable to insert artifact")
	}
	return nil
}

// SaveWorkflowFile Insert file in db and write it in data directory
func SaveWorkflowFile(art *sdk.WorkflowNodeRunArtifact, content io.ReadCloser) error {
	objectPath, err := objectstore.StoreArtifact(art, content)
	if err != nil {
		return sdk.WrapError(err, "SaveWorkflowFile> Cannot store artifact")
	}
	log.Debug("objectpath=%s\n", objectPath)
	art.ObjectPath = objectPath
	return nil
}

// SaveFile Insert file in db and write it in data directory
func SaveFile(db *gorp.DbMap, p *sdk.Pipeline, a *sdk.Application, art sdk.Artifact, content io.ReadCloser, e *sdk.Environment) error {
	tx, errB := db.Begin()
	if errB != nil {
		return sdk.WrapError(errB, "Cannot start transaction")
	}
	defer tx.Rollback()

	objectPath, errO := objectstore.StoreArtifact(&art, content)
	if errO != nil {
		return sdk.WrapError(errO, "SaveFile>Cannot store artifact")
	}
	log.Debug("objectpath=%s\n", objectPath)
	art.ObjectPath = objectPath
	if err := InsertArtifact(tx, p.ID, a.ID, e.ID, art); err != nil {
		return sdk.WrapError(err, "SaveFile> Cannot insert artifact in DB")
	}

	return tx.Commit()
}
