package authentication

import (
	"context"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

const (
	sessionMFAActivityDuration = 15 * time.Minute
)

func newSession(c *sdk.AuthConsumer, duration time.Duration) sdk.AuthSession {
	return sdk.AuthSession{
		ConsumerID: c.ID,
		ExpireAt:   time.Now().Add(duration),
	}
}

// NewSession returns a new session for a given auth consumer.
func NewSession(ctx context.Context, db gorpmapper.SqlExecutorWithTx, c *sdk.AuthConsumer, duration time.Duration) (*sdk.AuthSession, error) {
	s := newSession(c, duration)

	if err := InsertSession(ctx, db, &s); err != nil {
		return nil, err
	}

	return &s, nil
}

// NewSessionWithMFA returns a new session for a given auth consumer with MFA.
func NewSessionWithMFA(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, c *sdk.AuthConsumer, duration time.Duration) (*sdk.AuthSession, error) {
	s := newSession(c, duration)
	s.MFA = true

	if err := InsertSession(ctx, db, &s); err != nil {
		return nil, err
	}

	// Initialy set activity for new session
	if err := SetSessionActivity(store, s.ID); err != nil {
		return nil, err
	}

	return &s, nil
}

// CheckSession returns the session if valid for given id.
func CheckSession(ctx context.Context, db gorp.SqlExecutor, store cache.Store, sessionID string) (*sdk.AuthSession, error) {
	// Load the session from the id read in the claim
	s, err := LoadSessionByID(ctx, db, sessionID)
	if err != nil {
		return nil, sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrUnauthorized, "cannot load session for id: %s", sessionID))
	}
	if s == nil {
		log.Debug(ctx, "authentication.sessionMiddleware> no session found for id: %s", sessionID)
		return nil, sdk.WithStack(sdk.ErrUnauthorized)
	}
	if s.MFA {
		active, err := GetSessionActivity(store, s.ID)
		if err != nil {
			log.Error(ctx, "CheckSession> unable to get session %s activity: %v", s.ID, err)
		}
		if !active || err != nil {
			log.Debug(ctx, "authentication.sessionMiddleware> MFA session expired due to inactivity for id: %s", sessionID)
			if err := DeleteSessionByID(db, s.ID); err != nil {
				log.Error(ctx, "CheckSession> unable to delete session %s: %v", s.ID, err)
			}
			log.Debug(ctx, "CheckSession> MFA session %s deleted due to inactivity", s.ID)
			return nil, sdk.WithStack(sdk.ErrUnauthorized)
		}
		// If the session is valid we can update its activity
		if err := SetSessionActivity(store, s.ID); err != nil {
			return nil, err
		}
	}

	return s, nil
}

// NewSessionJWT generate a signed token for given auth session.
func NewSessionJWT(s *sdk.AuthSession, externalSessionID string) (string, error) {
	now := time.Now()
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodRS512, sdk.AuthSessionJWTClaims{
		ID:      s.ID,
		TokenID: externalSessionID,
		StandardClaims: jwt.StandardClaims{
			Issuer:    GetIssuerName(),
			Subject:   s.ConsumerID,
			Id:        s.ID,
			IssuedAt:  now.Unix(),
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
				log.Debug(ctx, "SessionCleaner> expired session %s deleted", s.ID)
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
				log.Debug(ctx, "SessionCleaner> corrupted session %s deleted", s.ID)
			}
		}
	}
}

// SetSessionActivity store activity in cache for given session.
func SetSessionActivity(store cache.Store, sessionID string) error {
	k := cache.Key("api", "session", "mfa", "activity", sessionID)
	if err := store.SetWithTTL(k, true, int(sessionMFAActivityDuration.Seconds())); err != nil {
		return err
	}
	return nil
}

// GetSessionActivity returns if given session is active.
func GetSessionActivity(store cache.Store, sessionID string) (bool, error) {
	k := cache.Key("api", "session", "mfa", "activity", sessionID)
	return store.Exist(k)
}
