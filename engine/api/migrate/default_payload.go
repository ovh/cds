package migrate

import (
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

	projs, errP := project.LoadAll(db, store, u)
	if errP != nil {
		log.Warning("DefaultPayloadMigration> Cannot load all project")
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

			wf, errWl := workflow.Load(tx, store, proj.Key, wfName, u, workflow.LoadOptions{})
			if errWl != nil {
				log.Warning("DefaultPayloadMigration> Cannot load workflow %s : %s", wfName, errWl)
				tx.Rollback()
				continue
			}

			errLock := workflow.LockNodeContext(tx, store, proj.Key, wf.Root.ID)
			if errLock != nil {
				log.Warning("DefaultPayloadMigration> Cannot lock node context for root %s : %s", wf.Root.Name, errLock)
				tx.Rollback()
				continue
			}

			haveRepoLinked := wf.Root != nil && wf.Root.Context != nil && wf.Root.Context.Application != nil && wf.Root.Context.Application.RepositoryFullname != ""
			if !haveRepoLinked {
				tx.Rollback()
				continue
			}

			m, errM := wf.Root.Context.DefaultPayloadToMap()
			if errM != nil {
				log.Warning("DefaultPayloadMigration> Cannot dump to map")
				tx.Rollback()
				continue
			}

			gitBranch, ok := m["git.branch"]
			if wf.Root.Context.HasDefaultPayload() && ok && gitBranch == "" {
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
		}
	}
}
