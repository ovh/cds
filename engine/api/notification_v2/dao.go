package notification_v2

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
)

func get(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) (*sdk.ProjectNotification, error) {
	var dbNotif dbProjectNotification
	found, err := gorpmapping.Get(ctx, db, q, &dbNotif)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.NewErrorFrom(sdk.ErrNotFound, "notification not found")
	}
	isValid, err := gorpmapping.CheckSignature(dbNotif, dbNotif.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "notification %s: data corrupted", dbNotif.ID)
		return nil, sdk.NewErrorFrom(sdk.ErrNotFound, "unable to find notification")
	}
	return &dbNotif.ProjectNotification, nil
}

func getAll(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]sdk.ProjectNotification, error) {
	var dbNotifs []dbProjectNotification
	if err := gorpmapping.GetAll(ctx, db, q, &dbNotifs); err != nil {
		return nil, err
	}

	notifs := make([]sdk.ProjectNotification, 0, len(dbNotifs))
	for _, n := range dbNotifs {
		isValid, err := gorpmapping.CheckSignature(n, n.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "notification %s: data corrupted", n.ID)
			continue
		}
		notifs = append(notifs, n.ProjectNotification)
	}

	return notifs, nil
}

func Insert(ctx context.Context, db gorpmapper.SqlExecutorWithTx, notif *sdk.ProjectNotification) error {
	notif.ID = sdk.UUID()
	notif.LastModified = time.Now()
	dbNotif := &dbProjectNotification{ProjectNotification: *notif}

	if err := gorpmapping.InsertAndSign(ctx, db, dbNotif); err != nil {
		return err
	}
	*notif = dbNotif.ProjectNotification
	return nil
}

func Update(ctx context.Context, db gorpmapper.SqlExecutorWithTx, notif *sdk.ProjectNotification) error {
	notif.LastModified = time.Now()
	dbNotif := &dbProjectNotification{ProjectNotification: *notif}
	if err := gorpmapping.UpdateAndSign(ctx, db, dbNotif); err != nil {
		return err
	}
	*notif = dbNotif.ProjectNotification
	return nil
}

func Delete(_ context.Context, db gorpmapper.SqlExecutorWithTx, notif *sdk.ProjectNotification) error {
	dbNotif := &dbProjectNotification{ProjectNotification: *notif}
	if err := gorpmapping.Delete(db, dbNotif); err != nil {
		return err
	}
	return nil
}

func LoadByName(ctx context.Context, db gorp.SqlExecutor, projectKey string, name string) (*sdk.ProjectNotification, error) {
	q := gorpmapping.NewQuery("SELECT * FROM project_notification WHERE project_key=$1 AND name=$2").Args(projectKey, name)
	return get(ctx, db, q)

}

func LoadAll(ctx context.Context, db gorp.SqlExecutor, projectKey string) ([]sdk.ProjectNotification, error) {
	q := gorpmapping.NewQuery("SELECT * FROM project_notification WHERE project_key=$1").Args(projectKey)
	return getAll(ctx, db, q)

}
