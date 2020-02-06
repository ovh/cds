package migrate

import (
	"context"
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// RefactorConsumerScope .
func RefactorConsumerScope(ctx context.Context, db *gorp.DbMap) error {
	log.Debug("migrate.RefactorConsumerScope> begin")
	defer func() {
		log.Debug("migrate.RefactorConsumerScope> end")
	}()

	rows, err := db.Query(`
    SELECT id
    FROM auth_consumer
    WHERE scope_details IS NULL
	`)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return sdk.WithStack(err)
	}
	defer rows.Close()

	var consumerIDs []string
	for rows.Next() {
		var consumerID string
		if err := rows.Scan(&consumerID); err != nil {
			return sdk.WithStack(err)
		}
		consumerIDs = append(consumerIDs, consumerID)
	}

	for _, id := range consumerIDs {
		if err := refactorConsumerScope(ctx, db, id); err != nil {
			log.Error(ctx, "migrate.RefactorConsumerScope> %+v", err)
		}
	}

	return nil
}

func refactorConsumerScope(ctx context.Context, db *gorp.DbMap, consumerID string) error {
	log.Info(ctx, "migrate.RefactorConsumerScope> trying to migrate consumer %s", consumerID)

	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}

	defer tx.Rollback() // nolint

	var consumer sdk.AuthConsumer
	var scopes sdk.AuthConsumerScopeSlice
	if err := tx.QueryRow(`
    SELECT
      id, name, description, parent_id, user_id, type, data, created, group_ids, invalid_group_ids, scopes, issued_at, disabled, warnings
    FROM auth_consumer
    WHERE scope_details IS NULL AND id = $1
    FOR UPDATE SKIP LOCKED
  `, consumerID).Scan(
		&consumer.ID, &consumer.Name, &consumer.Description, &consumer.ParentID, &consumer.AuthentifiedUserID,
		&consumer.Type, &consumer.Data, &consumer.Created, &consumer.GroupIDs, &consumer.InvalidGroupIDs,
		&scopes, &consumer.IssuedAt, &consumer.Disabled, &consumer.Warnings); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return sdk.WithStack(err)
	}

	log.Info(ctx, "migrate.RefactorConsumerScope> starting consumer migration %s", consumerID)

	for i := range scopes {
		consumer.ScopeDetails = append(consumer.ScopeDetails, sdk.AuthConsumerScopeDetail{
			Scope: scopes[i],
		})
	}

	if err := authentication.UpdateConsumer(ctx, tx, &consumer); err != nil {
		return sdk.WithStack(err)
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	log.Info(ctx, "migrate.RefactorConsumerScope> ending consumer migration %s", consumerID)

	return nil
}
