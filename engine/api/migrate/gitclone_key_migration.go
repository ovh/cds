package migrate

import (
	"regexp"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var badKey int64

// GitClonePrivateKey is temporary code
func GitClonePrivateKey(DBFunc func() *gorp.DbMap, store cache.Store) error {
	log.Info("GitClonePrivateKey> Begin")
	defer log.Info("GitClonePrivateKey> End with key errors %d", badKey)

	pipelines, err := action.GetPipelineUsingAction(DBFunc(), sdk.GitCloneAction)
	if err != nil {
		return err
	}
	db := DBFunc()

	for _, p := range pipelines {
		log.Debug("GitClonePrivateKey> Migrate %s/%s", p.ProjKey, p.PipName)

		tx, err := db.Begin()
		if err != nil {
			return sdk.WrapError(err, "GitClonePrivateKey> Cannot start transaction")
		}
		var id int64
		// Lock the job (action)
		if err := tx.QueryRow("select id from action where id = $1 for update nowait", p.ActionID).Scan(&id); err != nil {
			log.Debug("GitClonePrivateKey> unable to take lock on action table: %v", err)
			_ = tx.Rollback()
			continue
		}

		_ = id // we don't care about it
		if err := migrateActionGitClonePipeline(tx, store, p); err != nil {
			log.Error("GitClonePrivateKey> %v", err)
			_ = tx.Rollback()
			continue
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "GitClonePrivateKey> Cannot commit transaction")
		}

		log.Debug("GitClonePrivateKey> Migrate %s/%s DONE", p.ProjKey, p.PipName)
	}

	_, errEx := db.Exec(`
		UPDATE action_parameter
		  SET type = 'ssh-key', value = ''
		WHERE action_id = (
		  SELECT id
		    FROM action
		    WHERE name = 'GitClone' AND type = 'Builtin'
		) AND name = 'privateKey'`)

	return sdk.WrapError(errEx, "GitClonePrivateKey> cannot update action table builtin")
}

// migrateActionGitClonePipeline is the unitary function
func migrateActionGitClonePipeline(db gorp.SqlExecutor, store cache.Store, p action.PipelineUsingAction) error {
	pip, err := pipeline.LoadPipeline(db, p.ProjKey, p.PipName, true)
	if err != nil {
		return sdk.WrapError(err, "unable to load pipeline")
	}

	//Override the appname with the application in workflow node context if needed
	if p.AppName == "" && p.WorkflowName != "" {
		proj, err := project.Load(db, store, p.ProjKey, nil, project.LoadOptions.WithPlatforms)
		if err != nil {
			return err
		}
		w, err := workflow.Load(db, store, proj, p.WorkflowName, nil, workflow.LoadOptions{})
		if err != nil {
			return err
		}
		node := w.GetNodeByName(p.WorkflowNodeName)
		if node == nil {
			return sdk.ErrWorkflowNodeNotFound
		}
		if node.Context != nil && node.Context.Application != nil {
			p.AppName = node.Context.Application.Name
		}
	}

	for _, s := range pip.Stages {
		for _, j := range s.Jobs {
			var migrateJob bool
			for _, a := range j.Action.Actions {
				if a.Name == sdk.GitCloneAction {
					log.Debug("migrateActionGitClonePipeline> Migrate %s/%s/%s(%d)", p.ProjKey, p.PipName, j.Action.Name, j.Action.ID)
					migrateJob = true
					break
				}
			}
			if migrateJob {
				if err := migrateActionGitCloneJob(db, store, p.ProjKey, p.PipName, p.AppName, p.EnvID, j); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// migrateActionGitCloneJob is the unitary function
func migrateActionGitCloneJob(db gorp.SqlExecutor, store cache.Store, pkey, pipName, appName string, envID int64, j sdk.Job) error {
	mapReplacement := make(map[int]sdk.Action)

	//Load the first admin we can
	if anAdminID == 0 {
		users, err := user.LoadUsers(db)
		if err != nil {
			return err
		}
		for _, u := range users {
			if u.Admin {
				anAdminID = u.ID
				break
			}
		}
	}

	//Check all the steps of the job
	for i := range j.Action.Actions {
		step := &j.Action.Actions[i]
		log.Debug("migrateActionGitCloneJob>CheckJob> Checking step %s", step.Name)

		if step.Name == sdk.GitCloneAction {
			privateKey := sdk.ParameterFind(&step.Parameters, "privateKey")

			if privateKey.Value == "" || strings.HasPrefix(privateKey.Value, "proj-") || strings.HasPrefix(privateKey.Value, "app-") || strings.HasPrefix(privateKey.Value, "env-") {
				continue
			}

			switch {
			case strings.HasPrefix(privateKey.Value, "{{.cds.proj."):
				regx := regexp.MustCompile(`{{\.cds\.proj\.(.+)}}`)
				subMatch := regx.FindAllStringSubmatch(privateKey.Value, -1)
				if len(subMatch) > 0 && len(subMatch[0]) > 1 {
					//Load the project
					proj, err := project.Load(db, store, pkey, nil, project.LoadOptions.WithKeys)
					if err != nil {
						return err
					}
					kname := "proj-" + subMatch[0][1]
					if proj.GetSSHKey(kname) != nil {
						privateKey.Value = kname
						privateKey.Type = sdk.KeySSHParameter
					} else {
						badKey++
						log.Warning("migrateActionGitCloneJob> KEY NOT FOUND in project %s with key named %s", proj.Key, kname)
						continue
					}
				}
			case strings.HasPrefix(privateKey.Value, "{{.cds.env."):
				regx := regexp.MustCompile(`{{\.cds\.env\.(.+)}}`)
				subMatch := regx.FindAllStringSubmatch(privateKey.Value, -1)
				if len(subMatch) > 0 && len(subMatch[0]) > 1 && envID != 0 {
					env := sdk.Environment{ID: envID}
					if err := environment.LoadAllKeys(db, &env); err != nil {
						return err
					}
					kname := "env-" + subMatch[0][1]
					if env.GetSSHKey(kname) != nil {
						privateKey.Value = kname
						privateKey.Type = sdk.KeySSHParameter
					} else {
						badKey++
						log.Warning("migrateActionGitCloneJob> KEY NOT FOUND %s/%s in environment id %d with key named %s", pkey, pipName, env.ID, kname)
						continue
					}

				}
			case strings.HasPrefix(privateKey.Value, "{{.cds.app."):
				regx := regexp.MustCompile(`{{\.cds\.app\.(.+)}}`)
				subMatch := regx.FindAllStringSubmatch(privateKey.Value, -1)
				if len(subMatch) > 0 && len(subMatch[0]) > 1 && appName != "" {
					app, err := application.LoadByName(db, store, pkey, appName, nil, application.LoadOptions.WithKeys)
					if err != nil {
						return err
					}

					kname := "app-" + subMatch[0][1]
					if app.GetSSHKey(kname) != nil {
						privateKey.Value = kname
						privateKey.Type = sdk.KeySSHParameter
					} else {
						badKey++
						log.Warning("migrateActionGitCloneJob> KEY NOT FOUND in application %s/%s with key named %s", pkey, appName, kname)
						continue
					}
				}
			default:
				badKey++
				log.Warning("migrateActionGitCloneJob> Skipping %s/%s (%s) : can't find suitable key (%s)", pkey, pipName, j.Action.Name, privateKey.Value)
				continue
			}

			mapReplacement[i] = *step
			continue
		}
	}

	for i, a := range mapReplacement {
		j.Action.Actions[i] = a
	}

	//Update in database
	return action.UpdateActionDB(db, &j.Action, anAdminID)
}
