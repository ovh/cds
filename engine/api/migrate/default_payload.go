package migrate

import (
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// DefaultPayloadMigration useful to set default branch on default payload which git.branch is set to ""
func DefaultPayloadMigration(store cache.Store, DBFunc func() *gorp.DbMap, u *sdk.User) {
	db := DBFunc()

	projs, errP := project.LoadAll(db, store, u, project.LoadOptions.WithPlatforms)
	if errP != nil {
		log.Warning("DefaultPayloadMigration> Cannot load all project: %s", errP)
		return
	}

	for _, proj := range projs {
		workflowNames, err := workflow.LoadAllNames(db, proj.ID, u)
		if err != nil {
			log.Warning("DefaultPayloadMigration> Cannot load all workflow names : %s", err)
			continue
		}

		for _, wfName := range workflowNames {
			tx, errTx := db.Begin()
			if errTx != nil {
				log.Warning("DefaultPayloadMigration> Cannot start a transaction %s", errTx)
				continue
			}

			wf, errWl := workflow.Load(tx, store, &proj, wfName, u, workflow.LoadOptions{})
			if errWl != nil {
				log.Warning("DefaultPayloadMigration> Cannot load workflow %s : %s", wfName, errWl)
				tx.Rollback()
				continue
			}

			if wf == nil || wf.Root == nil {
				log.Warning("DefaultPayloadMigration> No ROOT linked to workflow %v", wf)
				tx.Rollback()
				continue
			}

			if errLock := workflow.LockNodeContext(tx, store, wf.Root.ID); errLock != nil {
				log.Warning("DefaultPayloadMigration> Cannot lock node context for root %s : %s", wf.Root.Name, errLock)
				tx.Rollback()
				continue
			}

			if !wf.Root.IsLinkedToRepo() {
				tx.Rollback()
				continue
			}

			m, errM := wf.Root.Context.DefaultPayloadToMap()
			if errM != nil {
				log.Warning("DefaultPayloadMigration> Cannot dump to map")
				tx.Rollback()
				continue
			}

			if wf.Root.Context.HasDefaultPayload() && m["git.branch"] == "" {
				defaultBranch := "master"
				projectVCSServer := repositoriesmanager.GetProjectVCSServer(&proj, wf.Root.Context.Application.VCSServer)
				if projectVCSServer != nil {
					client, errclient := repositoriesmanager.AuthorizedClient(tx, store, projectVCSServer)
					if errclient != nil {
						log.Warning("DefaultPayloadMigration> Cannot get authorized client")
						tx.Rollback()
						continue
					}

					branches, errBr := client.Branches(wf.Root.Context.Application.RepositoryFullname)
					if errBr != nil {
						log.Warning("DefaultPayloadMigration> Cannot get branches for %s", wf.Root.Context.Application.RepositoryFullname)
						tx.Rollback()
						continue
					}

					for _, branch := range branches {
						if branch.Default {
							defaultBranch = branch.DisplayID
							break
						}
					}
					m["git.branch"] = defaultBranch

					wf.Root.Context.DefaultPayload = m

					if err := workflow.UpdateNodeContext(tx, wf.Root.Context); err != nil {
						log.Warning("DefaultPayloadMigration> Cannot update node context : %s", err)
						tx.Rollback()
						continue
					}

					if err := tx.Commit(); err != nil {
						log.Warning("DefaultPayloadMigration> Cannot commit transaction : %s", err)
						tx.Rollback()
					}
				}
			} else {
				tx.Rollback()
			}

			migrateSchedulerPayload(db, store, &proj, wf)
		}
	}

	log.Info("DefaultPayloadMigration> Migration done")
}

func migrateSchedulerPayload(db *gorp.DbMap, store cache.Store, proj *sdk.Project, wf *sdk.Workflow) {
	for _, hook := range wf.Root.Hooks {
		if hook.WorkflowHookModel.Name == sdk.SchedulerModelName {
			tx, errTx := db.Begin()
			if errTx != nil {
				log.Warning("DefaultPayloadMigration> Cannot start a transaction for hooks : %s", errTx)
				continue
			}
			hookValue, errH := workflow.LoadAndLockHookByUUID(tx, hook.UUID)
			if errH != nil {
				log.Warning("DefaultPayloadMigration> Cannot LoadAndLockHooks : %s", errH)
				tx.Rollback()
				continue
			}

			hConfig := hookValue.Config.Values()
			if hConfig["payload"] != "" {
				payload := map[string]string{}
				if err := json.Unmarshal([]byte(hConfig["payload"]), &payload); err != nil {
					log.Warning("DefaultPayloadMigration> Cannot unmarshall payload to string for a scheduler : %s", err)
					tx.Rollback()
					continue
				}

				if payload["git.branch"] == "" {
					defaultBranch := "master"
					projectVCSServer := repositoriesmanager.GetProjectVCSServer(proj, wf.Root.Context.Application.VCSServer)
					if projectVCSServer != nil {
						client, errclient := repositoriesmanager.AuthorizedClient(tx, store, projectVCSServer)
						if errclient != nil {
							log.Warning("DefaultPayloadMigration> Cannot get authorized client")
							tx.Rollback()
							continue
						}

						branches, errBr := client.Branches(wf.Root.Context.Application.RepositoryFullname)
						if errBr != nil {
							log.Warning("DefaultPayloadMigration> Cannot get branches for %s", wf.Root.Context.Application.RepositoryFullname)
							tx.Rollback()
							continue
						}

						for _, branch := range branches {
							if branch.Default {
								defaultBranch = branch.DisplayID
								break
							}
						}
						payload["git.branch"] = defaultBranch
						if _, ok := payload[""]; ok {
							delete(payload, "")
						}

						payloadStr, errM := json.Marshal(&payload)
						if errM != nil {
							log.Warning("DefaultPayloadMigration> Cannot marshal hook config payload : %s", errM)
							tx.Rollback()
							continue
						}
						pl := hook.Config["payload"]
						pl.Value = string(payloadStr)
						hook.Config["payload"] = pl

						if err := workflow.UpdateHook(tx, &hook); err != nil {
							log.Warning("DefaultPayloadMigration> Cannot update hook : %s", err)
							tx.Rollback()
							continue
						}

						if err := tx.Commit(); err != nil {
							log.Warning("DefaultPayloadMigration> Cannot commit hook : %s", err)
							tx.Rollback()
							continue
						}
					}

				} else {
					tx.Rollback()
				}
			}
		}
	}
}
