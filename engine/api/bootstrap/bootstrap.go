package bootstrap

import (
	"strings"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/token"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// DefaultValues contains default user values for init DB
type DefaultValues struct {
	DefaultGroupName string
	SharedInfraToken string
}

//InitiliazeDB inits the database
func InitiliazeDB(defaultValues DefaultValues, DBFunc func() *gorp.DbMap) error {
	dbGorp := DBFunc()

	if err := group.CreateDefaultGroup(dbGorp, group.SharedInfraGroupName); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot setup default %s group", group.SharedInfraGroupName)
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

	return migratePersistentSessionToken(DBFunc)
}

func migratePersistentSessionToken(DBFunc func() *gorp.DbMap) error {
	tx, err := DBFunc().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("LOCK TABLE user_persistent_session IN ACCESS EXCLUSIVE MODE NOWAIT")
	count, err := tx.SelectInt("select count(1) from user_persistent_session")
	if err != nil {
		return err
	}

	if count != 0 {
		log.Debug("migratePersistentSession> Nothing to do")
		return nil
	}

	log.Info("migratePersistentSession> Begin")
	defer log.Info("migratePersistentSession> End")

	users, err := user.LoadUsers(tx)
	if err != nil {
		return err
	}

	for _, utmp := range users {
		u, err := user.LoadUserAndAuth(tx, utmp.Username)
		if err != nil {
			log.Error("migratePersistentSession> %v", err)
			break
		}

		for _, t := range u.Auth.Tokens {
			t.CreationDate = time.Now()
			t.LastConnectionDate = time.Now()
			t.UserID = u.ID
			t.Comment = "Automatic migration"
			if err := user.InsertPersistentSessionToken(tx, t); err != nil {
				log.Error("migratePersistentSession> %v", err)
				break
			}
		}

		u.Auth.Tokens = nil
		if err := user.UpdateUserAndAuth(tx, *u); err != nil {
			return err
		}
	}

	return tx.Commit()
}
