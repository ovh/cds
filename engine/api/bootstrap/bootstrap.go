package bootstrap

import (
	"database/sql"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/token"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//InitiliazeDB inits the database
func InitiliazeDB(defaultValues sdk.DefaultValues, DBFunc func() *gorp.DbMap) error {
	dbGorp := DBFunc()

	if err := group.CreateDefaultGroup(dbGorp, sdk.SharedInfraGroupName); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot setup default %s group", sdk.SharedInfraGroupName)
	}

	if strings.TrimSpace(defaultValues.DefaultGroupName) != "" {
		if err := group.CreateDefaultGroup(dbGorp, defaultValues.DefaultGroupName); err != nil {
			return sdk.WrapError(err, "InitiliazeDB> Cannot setup default %s group")
		}
	}

	if err := group.InitializeDefaultGroupName(dbGorp, defaultValues.DefaultGroupName); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot InitializeDefaultGroupName")
	}

	if err := token.Initialize(dbGorp, defaultValues.SharedInfraToken); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot InitializeDefaultGroupName")
	}

	if err := action.CreateBuiltinArtifactActions(dbGorp); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot setup builtin Artifact actions")
	}

	if err := action.CreateBuiltinActions(dbGorp); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot setup builtin actions")
	}

	if err := environment.CreateBuiltinEnvironments(dbGorp); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot setup builtin environments")
	}

	return nil
}

// VCSMigrate migrate stuffs to other stuffs
func VCSMigrate(db *gorp.DbMap, cache cache.Store) error {
	log.Info("VCSMigrate> Begin")
	defer log.Info("VCSMigrate> End")

	query := `
	select project.projectkey, repositories_manager_project.data, repositories_manager.name, repositories_manager.id
	from project, repositories_manager_project, repositories_manager 
	where project.id = repositories_manager_project.id_project
	and repositories_manager_project.id_repositories_manager = repositories_manager.id
	and project.vcs_servers is NULL
	`

	rows := []struct {
		Key  string         `db:"projectkey"`
		Name string         `db:"name"`
		Data sql.NullString `db:"data"`
		ID   int64          `db:"id"`
	}{}

	if _, err := db.Select(&rows, query); err != nil {
		return err
	}

	for _, r := range rows {
		log.Info("VCSMigrate> Migrating %s %s", r.Key, r.Name)
		tx, err := db.Begin()
		if err != nil {
			log.Error("VCSMigrate> unable to start transaction %v", err)
			continue
		}

		proj, err := project.Load(tx, cache, r.Key, nil, project.LoadOptions.WithApplications, project.LoadOptions.WithLockNoWait)
		if err != nil {
			log.Error("VCSMigrate> unable to load project %s: %v", r.Key, err)
			_ = tx.Rollback()
			continue
		}

		data := map[string]string{}
		if err := gorpmapping.JSONNullString(r.Data, &data); err != nil {
			log.Error("VCSMigrate> unable to unmarshall data %s: %v", r.Key, err)
			_ = tx.Rollback()
			continue
		}

		accessToken := data["access_token"]
		accessTokenSecret := data["access_token_secret"]

		//supportedName: stash.ovh.net + github
		vcsServerForProject := &sdk.ProjectVCSServer{
			Name: strings.Replace(r.Name, ".", "_", -1),
			Data: map[string]string{
				"token":  accessToken,
				"secret": accessTokenSecret,
			},
		}

		//Insert data for the project
		if err := repositoriesmanager.InsertForProject(tx, proj, vcsServerForProject); err != nil {
			log.Error("VCSMigrate> unable to migrate project %s: %v", r.Key, err)
			_ = tx.Rollback()
			continue
		}

		//Compute application
		for _, a := range proj.Applications {
			if a.RepositoryFullname != "" {
				appQuery := `update application set vcs_server = $3 where id = $1 and repositories_manager_id = $2`
				if _, err := tx.Exec(appQuery, a.ID, r.ID, vcsServerForProject.Name); err != nil {
					log.Error("VCSMigrate> unable to migrate application %s/%s: %v", r.Key, a.Name, err)
					_ = tx.Rollback()
					continue
				}
			}
		}

		if err := tx.Commit(); err != nil {
			log.Error("VCSMigrate> unable to commit transaction %v", err)
			_ = tx.Rollback()
			continue
		}
	}

	return nil
}
