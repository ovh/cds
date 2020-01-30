package migrate

import (
	"context"
	"database/sql"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func refactorGroup(ctx context.Context, db *gorp.DbMap) error {
	// First step
	// Migrate "group" entities to sign it
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}

	defer tx.Rollback() // nolint

	rows, err := tx.Query(`SELECT id, name FROM "group" WHERE sig IS NULL FOR UPDATE SKIP LOCKED`)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return sdk.WithStack(err)
	}
	defer rows.Close()

	var groups []*sdk.Group

	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return sdk.WithStack(err)
		}

		g := sdk.Group{
			ID:   id,
			Name: name,
		}

		groups = append(groups, &g)
	}

	for _, g := range groups {
		if err := group.Update(ctx, tx, g); err != nil {
			return sdk.WithStack(err)
		}
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	return nil
}

func refactorGroupMembership(ctx context.Context, db *gorp.DbMap) error {
	// Second step
	// Migrate data from table user_°group
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}

	defer tx.Rollback() // nolint

	rows, err := tx.Query(`
	SELECT group_user.group_id, authentified_user_migration.authentified_user_id, group_user.group_admin 
	FROM group_user
	JOIN authentified_user_migration ON authentified_user_migration.user_id = group_user.user_id
	WHERE authentified_user_migration.authentified_user_id NOT IN (
		SELECT DISTINCT authentified_user_id 
		FROM group_authentified_user
	)
	FOR UPDATE SKIP LOCKED
	`)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return sdk.WithStack(err)
	}
	defer rows.Close()

	var links []*group.LinkGroupUser

	for rows.Next() {
		var groupID int64
		var authentifiedUserID string
		var groupAdmin bool
		if err := rows.Scan(&groupID, &authentifiedUserID, &groupAdmin); err != nil {
			return sdk.WithStack(err)
		}

		var l = group.LinkGroupUser{
			GroupID:            groupID,
			AuthentifiedUserID: authentifiedUserID,
			Admin:              groupAdmin,
		}

		links = append(links, &l)
	}

	for _, l := range links {
		if err := group.InsertLinkGroupUser(ctx, tx, l); err != nil {
			return sdk.WithStack(err)
		}
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	return nil
}

// RefactorGroupMembership .
func RefactorGroupMembership(ctx context.Context, db *gorp.DbMap) error {
	log.Debug("migrate.RefactorGroupMembership> begin")
	defer func() {
		log.Debug("migrate.RefactorGroupMembership> end")
	}()

	if err := refactorGroup(ctx, db); err != nil {
		return err
	}

	if err := refactorGroupMembership(ctx, db); err != nil {
		return err
	}

	return nil
}
