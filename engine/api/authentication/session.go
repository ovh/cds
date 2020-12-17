package authentication

import (
	"context"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// NewSession returns a new session for a given auth consumer.
func NewSession(ctx context.Context, db gorpmapper.SqlExecutorWithTx, c *sdk.AuthConsumer, duration time.Duration, mfaEnable bool) (*sdk.AuthSession, error) {
	s := sdk.AuthSession{
		ConsumerID: c.ID,
		ExpireAt:   time.Now().Add(duration),
		MFA:        mfaEnable,
	}

	if err := InsertSession(ctx, db, &s); err != nil {
		return nil, err
	}

	return &s, nil
}

// CheckSession returns the session if valid for given id.
func CheckSession(ctx context.Context, db gorp.SqlExecutor, sessionID string) (*sdk.AuthSession, error) {
	// Load the session from the id read in the claim
	session, err := LoadSessionByID(ctx, db, sessionID)
	if err != nil {
		return nil, sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrUnauthorized, "cannot load session for id: %s", sessionID))
	}
	if session == nil {
		log.Debug("authentication.sessionMiddleware> no session found for id: %s", sessionID)
		return nil, sdk.WithStack(sdk.ErrUnauthorized)
	}

	return session, nil
}

// NewSessionJWT generate a signed token for given auth session.
func NewSessionJWT(s *sdk.AuthSession) (string, error) {
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodRS512, sdk.AuthSessionJWTClaims{
		ID:  s.ID,
		MFA: s.MFA,
		StandardClaims: jwt.StandardClaims{
			Issuer:    GetIssuerName(),
			Subject:   s.ConsumerID,
			Id:        s.ID,
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: s.ExpireAt.Unix(),
		},
	})
	return SignJWT(jwtToken)
}

// SessionCleaner must be run as a goroutine
func SessionCleaner(ctx context.Context, dbFunc func() *gorp.DbMap, tickerDuration time.Duration) {
	log.Info(ctx, "Initializing session cleaner...")
	db := dbFunc()
	tick := time.NewTicker(tickerDuration)
	tickCorruped := time.NewTicker(12 * time.Hour)
	defer tick.Stop()
	defer tickCorruped.Stop()

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "SessionCleaner> Exiting clean session: %v", ctx.Err())
				return
			}
		case <-tick.C:
			sessions, err := LoadExpiredSessions(ctx, db)
			if err != nil {
				log.Error(ctx, "SessionCleaner> unable to load expired sessions %v", err)
			}
			for _, s := range sessions {
				if err := DeleteSessionByID(db, s.ID); err != nil {
					log.Error(ctx, "SessionCleaner> unable to delete session %s: %v", s.ID, err)
				}
				log.Debug("SessionCleaner> expired session %s deleted", s.ID)
			}
		case <-tickCorruped.C:
			// This part of the goroutine should be remove in a next release
			sessions, err := UnsafeLoadCorruptedSessions(ctx, db)
			if err != nil {
				log.Error(ctx, "SessionCleaner> unable to load corrupted sessions %v", err)
			}
			for _, s := range sessions {
				if err := DeleteSessionByID(db, s.ID); err != nil {
					log.Error(ctx, "SessionCleaner> unable to delete session %s: %v", s.ID, err)
				}
				log.Debug("SessionCleaner> corrupted session %s deleted", s.ID)
			}
		}
	}
}
