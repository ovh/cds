package hook

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/spf13/viper"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

//ReceivedHook is a temporary struct to manage received hook
type ReceivedHook struct {
	URL        url.URL
	Data       []byte
	ProjectKey string
	Repository string
	Branch     string
	Hash       string
	Author     string
	Message    string
	UID        string
}

// HookLink format in stash/bitbucket
const HookLink = "/hook?uid=%s&project=%s&name=%s&branch=${refChange.name}&hash=${refChange.toHash}&message=${refChange.type}&author=${user.name}"

// InsertReceivedHook insert raw data received from public handler in database
func InsertReceivedHook(db gorp.SqlExecutor, link string, data string) error {
	query := `INSERT INTO received_hook (link, data) VALUES ($1, $2)`

	_, err := db.Exec(query, link, data)
	if err != nil {
		return err
	}

	return nil
}

// UpdateHook update the given hook
func UpdateHook(db gorp.SqlExecutor, h sdk.Hook) error {
	query := `UPDATE hook set pipeline_id=$1, kind=$2, host=$3, project=$4, repository=$5, application_id=$6, enabled=$7 WHERE id=$8`

	res, err := db.Exec(query, h.Pipeline.ID, h.Kind, h.Host, h.Project, h.Repository, h.ApplicationID, h.Enabled, h.ID)
	if err != nil {
		return err
	}
	nbRows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if nbRows != 1 {
		return sdk.ErrNoHook
	}
	return nil
}

// InsertHook add link between git repository and pipeline in database
func InsertHook(db gorp.SqlExecutor, h *sdk.Hook) error {
	query := `INSERT INTO hook (pipeline_id, kind, host, project, repository, application_id,enabled, uid) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`

	// Generate UID
	uid, err := generateHash()
	if err != nil {
		return err
	}
	h.UID = uid

	err = db.QueryRow(query, h.Pipeline.ID, h.Kind, h.Host, h.Project, h.Repository, h.ApplicationID, h.Enabled, h.UID).Scan(&h.ID)
	if err != nil {
		return err
	}

	return nil
}

// LoadHook loads a single hook
func LoadHook(db gorp.SqlExecutor, id int64) (sdk.Hook, error) {
	h := sdk.Hook{ID: id}
	query := `SELECT application_id, pipeline_id, kind, host, project, repository, enabled FROM hook WHERE id = $1`

	err := db.QueryRow(query, id).Scan(&h.ApplicationID, &h.Pipeline.ID, &h.Kind, &h.Host, &h.Project, &h.Repository, &h.Enabled)
	if err != nil {
		return h, err
	}

	return h, nil
}

//FindHook loads a hook from its attributes
func FindHook(db gorp.SqlExecutor, applicationID, pipelineID int64, kind, host, project, repository string) (sdk.Hook, error) {
	h := sdk.Hook{}
	query := `SELECT 	id, application_id, pipeline_id, kind, host, project, repository, uid
						FROM 		hook
						WHERE  	application_id=$1
						AND 		pipeline_id=$2
						AND 		kind=$3
						AND 		host=$4
						AND 		project=$5
						AND 		repository=$6`

	err := db.QueryRow(query, applicationID, pipelineID, kind, host, project, repository).Scan(&h.ID, &h.ApplicationID, &h.Pipeline.ID, &h.Kind, &h.Host, &h.Project, &h.Repository, &h.UID)
	if err != nil {
		return h, err
	}
	return h, nil
}

// DeleteHook removes hook from database
func DeleteHook(db gorp.SqlExecutor, id int64) error {
	query := `DELETE FROM hook WHERE id = $1`

	res, err := db.Exec(query, id)
	if err != nil {
		return err
	}
	nbRows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if nbRows != 1 {
		return sdk.ErrNoHook
	}

	return nil
}

// LoadApplicationHooks will load all hooks related to given application
func LoadApplicationHooks(db gorp.SqlExecutor, applicationID int64) ([]sdk.Hook, error) {
	hooks := []sdk.Hook{}
	query := `SELECT hook.id, hook.kind, hook.host, hook.project, hook.repository, hook.enabled, hook.uid, pipeline.id, pipeline.name
		  FROM hook
		  JOIN pipeline ON pipeline.id = hook.pipeline_id
		  WHERE application_id= $1
		  LIMIT 200`

	rows, err := db.Query(query, applicationID)
	if err != nil {
		return hooks, err
	}
	defer rows.Close()

	for rows.Next() {
		var h sdk.Hook
		h.ApplicationID = applicationID
		err = rows.Scan(&h.ID, &h.Kind, &h.Host, &h.Project, &h.Repository, &h.Enabled, &h.UID, &h.Pipeline.ID, &h.Pipeline.Name)
		if err != nil {
			return hooks, err
		}
		link := viper.GetString("api_url") + HookLink
		h.Link = fmt.Sprintf(link, h.UID, h.Project, h.Repository)
		hooks = append(hooks, h)
	}

	return hooks, nil
}

// LoadPipelineHooks will load all hooks related to given pipeline
func LoadPipelineHooks(db gorp.SqlExecutor, pipelineID int64, applicationID int64) ([]sdk.Hook, error) {
	query := `SELECT id, kind, host, project, repository, uid, enabled FROM hook WHERE pipeline_id = $1 AND application_id= $2`

	rows, err := db.Query(query, pipelineID, applicationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hooks []sdk.Hook
	for rows.Next() {
		var h sdk.Hook
		h.Pipeline.ID = pipelineID
		h.ApplicationID = applicationID
		if err = rows.Scan(&h.ID, &h.Kind, &h.Host, &h.Project, &h.Repository, &h.UID, &h.Enabled); err != nil {
			return nil, err
		}
		link := viper.GetString("api_url") + HookLink
		h.Link = fmt.Sprintf(link, h.UID, h.Project, h.Repository)
		hooks = append(hooks, h)
	}

	return hooks, nil
}

// LoadHooks related to given repository
func LoadHooks(db gorp.SqlExecutor, project string, repository string) ([]sdk.Hook, error) {
	query := `SELECT id, pipeline_id, application_id, kind, host, enabled, uid FROM hook WHERE project = $1 AND repository = $2`

	rows, err := db.Query(query, project, repository)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hooks []sdk.Hook
	for rows.Next() {
		var h sdk.Hook
		h.Project = project
		h.Repository = repository
		err = rows.Scan(&h.ID, &h.Pipeline.ID, &h.ApplicationID, &h.Kind, &h.Host, &h.Enabled, &h.UID)
		if err != nil {
			return nil, err
		}
		hooks = append(hooks, h)
	}

	return hooks, nil
}

func generateHash() (string, error) {
	size := 128
	bs := make([]byte, size)
	_, err := rand.Read(bs)
	if err != nil {
		log.Critical("generateID: rand.Read failed: %s\n", err)
		return "", err
	}
	str := hex.EncodeToString(bs)
	token := []byte(str)[0:size]

	log.Debug("generateID: new generated id: %s\n", token)
	return string(token), nil
}

// DeleteBranchBuilds deletes all builds related to given branch in given applications in pipeline_build
func DeleteBranchBuilds(db gorp.SqlExecutor, hooks []sdk.Hook, branch string) error {

	for i := range hooks {
		err := deleteBranchBuilds(db, hooks[i].ApplicationID, branch)
		if err != nil {
			log.Warning("DeleteBranchBuilds> Cannot delete branch builds for branch %s in %d\n", branch, hooks[i].ApplicationID)
		}
	}
	return nil
}

func deleteBranchBuilds(db gorp.SqlExecutor, appID int64, branch string) error {

	pbs, errPB := pipeline.LoadPipelineBuildByApplicationAndBranch(db, appID, branch)
	if errPB != nil {
		return errPB
	}

	// Disabled building worker
	for _, pb := range pbs {
		if pb.Status != sdk.StatusBuilding {
			continue
		}
		for _, s := range pb.Stages {
			if s.Status != sdk.StatusBuilding {
				continue
			}
			for _, pbJob := range s.PipelineBuildJobs {
				if err := worker.DisableBuildingWorker(db, pbJob.ID); err != nil {
					log.Warning("deleteBranchBuilds> Cannot disabled worker")
					return err
				}
			}
		}

		// Stop building pipeline
		if err := pipeline.StopPipelineBuild(db, &pb); err != nil {
			log.Warning("deleteBranchBuilds> Cannot stop pipeline")
			continue
		}

	}
	// Now select all related build in pipeline build
	query := `SELECT id FROM pipeline_build WHERE vcs_changes_branch = $1 AND application_id = $2`
	rows, err := db.Query(query, branch, appID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return err
		}

		if err := pipeline.DeletePipelineBuildByID(db, id); err != nil {
			log.Warning("deleteBranchBuilds> Cannot delete PipelineBuild %d: %s\n", id, err)
		}
	}
	rows.Close()

	return nil
}

// CreateHook in CDS db + repo manager webhook
func CreateHook(tx gorp.SqlExecutor, projectKey string, rm *sdk.RepositoriesManager, repoFullName string, application *sdk.Application, pipeline *sdk.Pipeline) (*sdk.Hook, error) {
	client, err := repositoriesmanager.AuthorizedClient(tx, projectKey, rm.Name)
	if err != nil {
		log.Warning("addHookOnRepositoriesManagerHandler> Cannot get client got %s %s : %s", projectKey, rm.Name, err)
		return nil, err
	}

	t := strings.Split(repoFullName, "/")
	if len(t) != 2 {
		log.Warning("CreateHook> Wrong repo fullname %s.", repoFullName)
		return nil, fmt.Errorf("CreateHook> Wrong repo fullname %s.", repoFullName)
	}

	var h sdk.Hook

	h, err = FindHook(tx, application.ID, pipeline.ID, string(rm.Type), rm.URL, t[0], t[1])
	if err == sql.ErrNoRows {
		h = sdk.Hook{
			Pipeline:      *pipeline,
			ApplicationID: application.ID,
			Kind:          string(rm.Type),
			Host:          rm.URL,
			Project:       t[0],
			Repository:    t[1],
			Enabled:       true,
		}
		err = InsertHook(tx, &h)
		if err != nil {
			log.Warning("addHookOnRepositoriesManagerHandler> Cannot insert hook: %s", err)
			return nil, err
		}
	} else if err != nil {
		log.Warning("addHookOnRepositoriesManagerHandler> Cannot get hook: %s", err)
		return nil, err
	}

	s := viper.GetString("api_url") + HookLink
	link := fmt.Sprintf(s, h.UID, t[0], t[1])

	h.Link = link

	err = client.CreateHook(repoFullName, link)
	if err != nil {
		log.Warning("addHookOnRepositoriesManagerHandler> Cannot create hook on stash: %s", err)
		return nil, err
	}
	return &h, nil
}

//Recovery try to recovers hook in case of error
func Recovery(h ReceivedHook, err error) {
	log.Debug("hook.Recovery> %s", h.Repository)
	switch err.(type) {
	case sdk.Error:
		log.Notice("hook.Recovery> %s is not handled", h.Repository)
		return
	default:

	}
	switch s := err.Error(); s {
	case
		"database not available",
		"sql: database is closed",
		"sql: connection returned that was never out",
		"sql: Transaction has already been committed or rolled back",
		"sql: statement is closed",
		"sql: Rows are closed",
		"sql: no Rows available",
		"database/sql: internal sentinel error: conn is closed",
		"database/sql: internal sentinel error: conn is busy":
		log.Debug("hook.Recovery> Save %s/%s/%s for recover", h.ProjectKey, h.Repository, h.Hash)
	default:
		log.Notice("hook.Recovery> %s is not handled", s)
		return
	}

	cache.Enqueue("hook:recovery", h)

	return
}
