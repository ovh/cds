package authentication

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	jwt "github.com/golang-jwt/jwt"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

const (
	sessionMFAActivityDuration = 15 * time.Minute
)

func newSession(c *sdk.AuthUserConsumer, duration time.Duration) sdk.AuthSession {
	return sdk.AuthSession{
		ConsumerID: c.ID,
		ExpireAt:   time.Now().Add(duration),
	}
}

// NewSession returns a new session for a given auth consumer.
func NewSession(ctx context.Context, db gorpmapper.SqlExecutorWithTx, c *sdk.AuthUserConsumer, duration time.Duration) (*sdk.AuthSession, error) {
	s := newSession(c, duration)

	if err := InsertSession(ctx, db, &s); err != nil {
		return nil, err
	}

	return &s, nil
}

// NewSessionWithMFA returns a new session for a given auth consumer with MFA.
func NewSessionWithMFA(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, c *sdk.AuthUserConsumer, duration time.Duration) (*sdk.AuthSession, error) {
	return NewSessionWithMFACustomDuration(ctx, db, store, c, duration, sessionMFAActivityDuration)
}

// NewSessionWithMFACustomDuration returns a new session for a given auth consumer with MFA and custom MFA duration.
func NewSessionWithMFACustomDuration(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, c *sdk.AuthUserConsumer, duration, durationMFA time.Duration) (*sdk.AuthSession, error) {
	s := newSession(c, duration)
	s.MFA = true

	if err := InsertSession(ctx, db, &s); err != nil {
		return nil, err
	}

	// Initialy set activity for new session
	if err := SetSessionActivity(store, durationMFA, s.ID); err != nil {
		return nil, err
	}

	return &s, nil
}

// CheckSessionWithCustomMFADuration returns the session if valid for given id.
func CheckSessionWithCustomMFADuration(ctx context.Context, db gorp.SqlExecutor, store cache.Store, sessionID string, durationMFA time.Duration) (*sdk.AuthSession, error) {
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
		active, _, err := GetSessionActivity(store, s.ID)
		if err != nil {
			log.Error(ctx, "CheckSession> unable to get session %s activity: %v", s.ID, err)
		}
		if !active || err != nil {
			log.Info(ctx, "authentication.sessionMiddleware> Session MFA expired due to inactivity for id: %s", sessionID)
			s.MFA = false
		} else {
			// If the session is valid we can update its activity
			if err := SetSessionActivity(store, durationMFA, s.ID); err != nil {
				return nil, err
			}
		}
	}

	return s, nil
}

// CheckSession returns the session if valid for given id.
func CheckSession(ctx context.Context, db gorp.SqlExecutor, store cache.Store, sessionID string) (*sdk.AuthSession, error) {
	return CheckSessionWithCustomMFADuration(ctx, db, store, sessionID, sessionMFAActivityDuration)
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
func SetSessionActivity(store cache.Store, durationMFA time.Duration, sessionID string) error {
	k := cache.Key("api", "session", "mfa", "activity", sessionID)
	if err := store.SetWithTTL(k, time.Now().UnixNano(), int(durationMFA.Seconds())); err != nil {
		return err
	}
	return nil
}

// GetSessionActivity returns if given session is active.
func GetSessionActivity(store cache.Store, sessionID string) (exists bool, lastActivity time.Time, err error) {
	k := cache.Key("api", "session", "mfa", "activity", sessionID)
	var lastActivityUnixNano int64
	exists, err = store.Get(k, &lastActivityUnixNano)
	if err != nil {
		return
	}
	if !exists {
		return
	}
	lastActivity = time.Unix(0, lastActivityUnixNano)
	return
}
