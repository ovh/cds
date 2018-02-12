package hook

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var apiURL string

// Init initialize the hook package
func Init(url string) {
	apiURL = url
}

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
	query := `INSERT INTO hook (pipeline_id, kind, host, project, repository, application_id, enabled, uid) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`

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
	query := `SELECT uid, application_id, pipeline_id, kind, host, project, repository, enabled FROM hook WHERE id = $1`

	err := db.QueryRow(query, id).Scan(&h.UID, &h.ApplicationID, &h.Pipeline.ID, &h.Kind, &h.Host, &h.Project, &h.Repository, &h.Enabled)
	if err != nil {
		return h, err
	}

	return h, nil
}

// LoadHookByUID loads a single hook
func LoadHookByUID(db gorp.SqlExecutor, uid string) (sdk.Hook, error) {
	h := sdk.Hook{UID: uid}
	query := `SELECT id, application_id, pipeline_id, kind, host, project, repository, enabled FROM hook WHERE uuid = $1`

	err := db.QueryRow(query, uid).Scan(&h.ID, &h.ApplicationID, &h.Pipeline.ID, &h.Kind, &h.Host, &h.Project, &h.Repository, &h.Enabled)
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
		link := apiURL + HookLink
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
		link := apiURL + HookLink
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
		log.Error("generateID: rand.Read failed: %s\n", err)
		return "", err
	}
	str := hex.EncodeToString(bs)
	token := []byte(str)[0:size]

	log.Debug("generateID: new generated id: %s\n", token)
	return string(token), nil
}

// DeleteBranchBuilds deletes all builds related to given branch in given applications in pipeline_build
func DeleteBranchBuilds(db gorp.SqlExecutor, store cache.Store, hooks []sdk.Hook, branch string) error {
	for i := range hooks {
		if err := pipeline.DeleteBranchBuilds(db, store, hooks[i].ApplicationID, branch); err != nil {
			return sdk.WrapError(err, "DeleteBranchBuilds> Cannot delete branch builds for branch %s in %d", branch, hooks[i].ApplicationID)
		}
	}
	return nil
}

// CreateHook in CDS db + repo manager webhook
func CreateHook(tx gorp.SqlExecutor, store cache.Store, proj *sdk.Project, rm, repoFullName string, application *sdk.Application, pipeline *sdk.Pipeline) (*sdk.Hook, error) {
	server := repositoriesmanager.GetProjectVCSServer(proj, rm)
	if server == nil {
		return nil, fmt.Errorf("Unable to find repository manager")
	}
	client, err := repositoriesmanager.AuthorizedClient(tx, store, server)
	if err != nil {
		return nil, sdk.WrapError(err, "CreateHook> Cannot get client, got  %s %s", proj.Key, rm)
	}

	//Check if the webhooks if disabled
	if info, err := repositoriesmanager.GetWebhooksInfos(client); err != nil {
		return nil, err
	} else if !info.WebhooksSupported || info.WebhooksDisabled {
		return nil, sdk.WrapError(sdk.NewError(sdk.ErrForbidden, fmt.Errorf("Webhooks are not supported on %s", server.Name)), "CreateHook>")
	}

	t := strings.Split(repoFullName, "/")
	if len(t) != 2 {
		return nil, sdk.WrapError(fmt.Errorf("CreateHook> Wrong repo fullname %s", repoFullName), "")
	}

	var h sdk.Hook

	h, err = FindHook(tx, application.ID, pipeline.ID, rm, rm, t[0], t[1])
	if err == sql.ErrNoRows {
		h = sdk.Hook{
			Pipeline:      *pipeline,
			ApplicationID: application.ID,
			Kind:          rm,
			Host:          rm,
			Project:       t[0],
			Repository:    t[1],
			Enabled:       true,
		}
		if err := InsertHook(tx, &h); err != nil {
			return nil, sdk.WrapError(err, "CreateHook> Cannot insert hook")
		}
	} else if err != nil {
		return nil, sdk.WrapError(err, "CreateHook> Cannot get hook")
	}

	s := apiURL + HookLink
	h.Link = fmt.Sprintf(s, h.UID, t[0], t[1])

	hook := sdk.VCSHook{
		Method:   "POST",
		URL:      h.Link,
		Workflow: false,
	}

	log.Info("CreateHook> will create %+v", hook)

	if err := client.CreateHook(repoFullName, &hook); err != nil {
		log.Warning("Cannot create hook on repository manager: %s", err)
		if strings.Contains(err.Error(), "Not yet implemented") {
			return nil, sdk.WrapError(sdk.ErrNotImplemented, "CreateHook> Cannot create hook on repository manager")
		}
		if err := DeleteHook(tx, h.ID); err != nil {
			return nil, sdk.WrapError(err, "CreateHook> Cannot rollback hook creation")
		}
	}
	return &h, nil
}

//Recovery try to recovers hook in case of error
func Recovery(store cache.Store, h ReceivedHook, err error) {
	log.Debug("hook.Recovery> %s", h.Repository)
	switch err.(type) {
	case sdk.Error:
		log.Info("hook.Recovery> %s is not handled", h.Repository)
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
		log.Info("hook.Recovery> %s is not handled", s)
		return
	}

	store.Enqueue("hook:recovery", h)

	return
}
