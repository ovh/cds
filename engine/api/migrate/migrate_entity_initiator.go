package migrate

import (
	"context"
	"fmt"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

const MigrateEntityInitiatorName = "MigrateEntityInitiator"

// MigrateEntityInitiator gives an owner to the entities written before the initiator column existed,
// head entities first, and re-signs them.
// The overall progress is the number of rows still having a NULL initiator.
func MigrateEntityInitiator(ctx context.Context, db *gorp.DbMap) error {
	ids, err := entity.LoadIDsWithoutInitiator(ctx, db)
	if err != nil {
		return err
	}
	log.Info(ctx, "%s: %d entities to migrate", MigrateEntityInitiatorName, len(ids))
	return migrateEntityInitiators(ctx, db, ids)
}

func migrateEntityInitiators(ctx context.Context, db *gorp.DbMap, ids []string) error {
	owners := newLegacyOwners()
	var migrated, skipped, failed int
	for i, id := range ids {
		done, err := migrateEntityInitiator(ctx, db, id, owners)
		switch {
		case err != nil:
			failed++
			log.Error(ctx, "%s: entity %s: %v", MigrateEntityInitiatorName, id, err)
		case done:
			migrated++
		default:
			skipped++
		}
		if (i+1)%1000 == 0 {
			log.Info(ctx, "%s: %d/%d entities processed", MigrateEntityInitiatorName, i+1, len(ids))
		}
	}
	log.Info(ctx, "%s: %d migrated, %d skipped, %d failed", MigrateEntityInitiatorName, migrated, skipped, failed)
	if failed > 0 {
		return fmt.Errorf("%d entities could not be migrated, run the migration again to retry them", failed)
	}
	return nil
}

// migrateEntityInitiator rebuilds the owner of one entity from its user_id column, with the same user
// snapshot as a run initiator. It returns false when there is nothing to do: the row is gone,
// unreadable, locked by another writer, or already carries an owner.
func migrateEntityInitiator(ctx context.Context, db *gorp.DbMap, id string, owners *legacyOwners) (bool, error) {
	tx, err := db.Begin()
	if err != nil {
		return false, sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	e, err := entity.LoadAndLockByID(ctx, tx, id)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	if e.Initiator != nil {
		return false, nil
	}
	e.Initiator, err = owners.owner(ctx, tx, e.DeprecatedUserID)
	if err != nil {
		return false, err
	}
	if err := entity.Update(ctx, tx, e); err != nil {
		return false, err
	}
	return true, sdk.WithStack(tx.Commit())
}

// legacyOwners builds owners from the user_id column written by former versions, loading each user
// once per migration run.
type legacyOwners struct {
	users map[string]*sdk.V2Initiator
}

func newLegacyOwners() *legacyOwners {
	return &legacyOwners{users: make(map[string]*sdk.V2Initiator)}
}

// owner returns nobody when user_id is unset, otherwise a copy of the user's owner, with its snapshot.
// A deleted user keeps its id with an empty snapshot.
func (c *legacyOwners) owner(ctx context.Context, db gorp.SqlExecutor, userID *string) (*sdk.V2Initiator, error) {
	if userID == nil || *userID == "" {
		return &sdk.V2Initiator{}, nil
	}
	owner, known := c.users[*userID]
	if !known {
		u, err := user.LoadByID(ctx, db, *userID, user.LoadOptions.WithContacts)
		switch {
		case err == nil:
			owner = &sdk.V2Initiator{UserID: u.ID, User: u.Initiator()}
		case sdk.ErrorIs(err, sdk.ErrUserNotFound) || sdk.ErrorIs(err, sdk.ErrNotFound):
			owner = &sdk.V2Initiator{UserID: *userID, User: &sdk.V2InitiatorUser{}}
		default:
			return nil, err
		}
		c.users[*userID] = owner
	}
	return owner.EntityOwner(), nil
}
