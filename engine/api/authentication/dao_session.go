package authentication

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getSessions(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadSessionOptionFunc) ([]sdk.AuthSession, error) {
	ss := []authSession{}

	if err := gorpmapping.GetAll(ctx, db, q, &ss); err != nil {
		return nil, sdk.WrapError(err, "cannot get auth sessions")
	}

	// Check signature of data, if invalid do not return it
	verifiedSessions := make([]*sdk.AuthSession, 0, len(ss))
	for i := range ss {
		isValid, err := gorpmapping.CheckSignature(ss[i], ss[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error("authentication.getSessions> auth session %s data corrupted", ss[i].ID)
			continue
		}
		verifiedSessions = append(verifiedSessions, &ss[i].AuthSession)
	}

	if len(verifiedSessions) > 0 {
		for i := range opts {
			if err := opts[i](ctx, db, verifiedSessions...); err != nil {
				return nil, err
			}
		}
	}

	sessions := make([]sdk.AuthSession, len(verifiedSessions))
	for i := range verifiedSessions {
		sessions[i] = *verifiedSessions[i]
	}

	return sessions, nil
}

func getSession(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadSessionOptionFunc) (*sdk.AuthSession, error) {
	var session authSession

	found, err := gorpmapping.Get(ctx, db, q, &session)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get auth session")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	isValid, err := gorpmapping.CheckSignature(session, session.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error("authentication.getSession> auth session %s data corrupted", session.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	as := session.AuthSession

	for i := range opts {
		if err := opts[i](ctx, db, &as); err != nil {
			return nil, err
		}
	}

	return &as, nil
}

// LoadSessionByID returns an auth session from database.
func LoadSessionByID(ctx context.Context, db gorp.SqlExecutor, id string, opts ...LoadSessionOptionFunc) (*sdk.AuthSession, error) {
	query := gorpmapping.NewQuery("SELECT * FROM auth_session WHERE id = $1").Args(id)
	return getSession(ctx, db, query, opts...)
}

// InsertSession in database.
func InsertSession(db gorp.SqlExecutor, as *sdk.AuthSession) error {
	as.ID = sdk.UUID()
	as.Created = time.Now()
	s := authSession{AuthSession: *as}
	if err := gorpmapping.InsertAndSign(db, &s); err != nil {
		return sdk.WrapError(err, "unable to insert auth session")
	}
	*as = s.AuthSession
	return nil
}

// UpdateSession in database.
func UpdateSession(db gorp.SqlExecutor, as *sdk.AuthSession) error {
	s := authSession{AuthSession: *as}
	if err := gorpmapping.UpdatetAndSign(db, &s); err != nil {
		return sdk.WrapError(err, "unable to update auth session with id: %s", s.ID)
	}
	*as = s.AuthSession
	return nil
}

// DeleteSessionByID removes a auth session in database for given id.
func DeleteSessionByID(db gorp.SqlExecutor, id string) error {
	_, err := db.Exec("DELETE FROM auth_session WHERE id = $1", id)
	return sdk.WrapError(err, "unable to delete auth session with id %s", id)
}
