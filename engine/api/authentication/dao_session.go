package authentication

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gorpmapping"
	"github.com/ovh/cds/sdk/log"
)

// UnsafeLoadCorruptedSessions should not be used
func UnsafeLoadCorruptedSessions(ctx context.Context, db gorp.SqlExecutor) ([]sdk.AuthSession, error) {
	ss := []authSession{}
	q := gorpmapping.NewQuery(`SELECT *
	FROM auth_session
	ORDER BY created ASC`)
	if err := gorpmapping.GetAll(ctx, db, q, &ss); err != nil {
		return nil, sdk.WrapError(err, "cannot get auth sessions")
	}

	// Check signature of data, to get only invalid signatures
	corruptedSessions := make([]sdk.AuthSession, 0, len(ss))
	for i := range ss {
		isValid, _ := gorpmapping.CheckSignature(ss[i], ss[i].Signature)
		// If the signature is valid, to not consider the session as corrupted
		if isValid || ss[i].ID == "" {
			continue
		}
		corruptedSessions = append(corruptedSessions, ss[i].AuthSession)
	}
	log.Info(ctx, "authentication.UnsafeLoadCorruptedSessions> %d corrupted sessions", len(corruptedSessions))
	return corruptedSessions, nil
}

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
			log.Error(ctx, "authentication.getSessions> auth session %s data corrupted", ss[i].ID)
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
		log.Error(ctx, "authentication.getSession> auth session %s data corrupted", session.ID)
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

// LoadExpiredSessions returns all expired session
func LoadExpiredSessions(ctx context.Context, db gorp.SqlExecutor, opts ...LoadSessionOptionFunc) ([]sdk.AuthSession, error) {
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM auth_session
		WHERE expire_at < $1
		ORDER BY created ASC
	`).Args(time.Now())
	return getSessions(ctx, db, query, opts...)
}

// LoadSessionsByConsumerIDs returns all auth sessions from database for given consumer ids.
func LoadSessionsByConsumerIDs(ctx context.Context, db gorp.SqlExecutor, consumerIDs []string, opts ...LoadSessionOptionFunc) ([]sdk.AuthSession, error) {
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM auth_session
		WHERE consumer_id = ANY(string_to_array($1, ',')::text[])
		ORDER BY created ASC
	`).Args(gorpmapping.IDStringsToQueryString(consumerIDs))
	return getSessions(ctx, db, query, opts...)
}

// LoadSessionByID returns an auth session from database.
func LoadSessionByID(ctx context.Context, db gorp.SqlExecutor, id string, opts ...LoadSessionOptionFunc) (*sdk.AuthSession, error) {
	query := gorpmapping.NewQuery("SELECT * FROM auth_session WHERE id = $1").Args(id)
	return getSession(ctx, db, query, opts...)
}

// InsertSession in database.
func InsertSession(ctx context.Context, db gorpmapping.SqlExecutorWithTx, as *sdk.AuthSession) error {
	as.ID = sdk.UUID()
	as.Created = time.Now()
	s := authSession{AuthSession: *as}
	if err := gorpmapping.InsertAndSign(ctx, db, &s); err != nil {
		return sdk.WrapError(err, "unable to insert auth session")
	}
	*as = s.AuthSession
	return nil
}

// UpdateSession in database.
func UpdateSession(ctx context.Context, db gorpmapping.SqlExecutorWithTx, as *sdk.AuthSession) error {
	s := authSession{AuthSession: *as}
	if err := gorpmapping.UpdateAndSign(ctx, db, &s); err != nil {
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
