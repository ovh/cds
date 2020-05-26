package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// RefactorProjectVCSServers .
func RefactorProjectVCSServers(ctx context.Context, db *gorp.DbMap) error {
	query := `
	SELECT id 
	FROM project 
	WHERE vcs_servers IS NOT NULL 
	AND NOT EXISTS (
		SELECT 1 
		FROM project_vcs_server_link
		WHERE project.id = project_vcs_server_link.project_id
	)`
	rows, err := db.Query(query)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return sdk.WithStack(err)
	}

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close() // nolint
			return sdk.WithStack(err)
		}
		ids = append(ids, id)
	}

	if err := rows.Close(); err != nil {
		return sdk.WithStack(err)
	}

	var mError = new(sdk.MultiError)
	for _, id := range ids {
		if err := refactorProjectVCSServers(ctx, db, id); err != nil {
			mError.Append(err)
			log.Error(ctx, "migrate.RefactorProjectVCSServers> unable to migrate project %d: %v", id, err)
		}
	}

	if mError.IsEmpty() {
		return nil
	}
	return mError
}

func refactorProjectVCSServers(ctx context.Context, db *gorp.DbMap, id int64) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}

	defer tx.Rollback() // nolint

	query := `SELECT * FROM project WHERE id = $1 FOR UPDATE SKIP LOCKED`
	if _, err := tx.Exec(query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return sdk.WithStack(err)
	}

	proj, err := project.LoadByID(tx, id)
	if err != nil {
		return sdk.WithStack(err)
	}

	log.Info(ctx, "migrate.refactorProjectVCSServers> project %s (%d) started migration", proj.Name, proj.ID)

	for _, vcsServer := range proj.DeprecatedVCSServers {
		newVCSServer := &sdk.ProjectVCSServerLink{
			ProjectID: id,
			Name:      vcsServer.Name,
			Username:  vcsServer.Username,
		}
		for k, v := range vcsServer.Data {
			log.Debug("setting %s for %s", k, vcsServer.Name)
			newVCSServer.Set(k, v)
		}
		if err := repositoriesmanager.InsertProjectVCSServerLink(ctx, tx, newVCSServer); err != nil {
			return err
		}
	}

	// Now checks that the migration is fine
	allMigratedVCSServers, err := repositoriesmanager.LoadAllProjectVCSServerLinksByProjectID(ctx, tx, id)
	if err != nil {
		return err
	}

	if len(proj.DeprecatedVCSServers) != len(allMigratedVCSServers) {
		return sdk.WithStack(errors.New("not the same number of vcs_server :("))
	}

	for _, vcsServer := range proj.DeprecatedVCSServers {
		var found bool
		for i := range allMigratedVCSServers {
			migratedVCSServer := &allMigratedVCSServers[i]
			if vcsServer.Name == migratedVCSServer.Name {
				found = true
				if vcsServer.Username != migratedVCSServer.Username {
					return sdk.WithStack(fmt.Errorf("assertion failed on username: %s %s", vcsServer.Username, migratedVCSServer.Username))
				}
				newData, err := repositoriesmanager.LoadProjectVCSServerLinksData(ctx, tx, migratedVCSServer.ID, gorpmapping.GetOptions.WithDecryption)
				if err != nil {
					return err
				}
				log.Debug("newData: %+v", newData)
				migratedVCSServer.ProjectVCSServerLinkData = newData
				for k, v := range vcsServer.Data {
					newValue, foundValue := migratedVCSServer.Get(k)
					if !foundValue {
						return sdk.WithStack(fmt.Errorf("assertion failed: missing value %s", k))
					}
					if newValue != v {
						return sdk.WithStack(fmt.Errorf("assertion failed: value %s doesn't match (%s - %s)", k, v, newValue))
					}
				}
			}
		}
		if !found {
			return sdk.WithStack(errors.New("not missing vcs_server"))
		}
	}

	log.Info(ctx, "migrate.refactorProjectVCSServers> project %s (%d) migrated", proj.Name, proj.ID)

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
