package authentication

import (
	"context"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// NewSession returns a new session for a given auth consumer.
func NewSession(db gorp.SqlExecutor, c *sdk.AuthConsumer, duration time.Duration, mfaEnable bool) (*sdk.AuthSession, error) {
	s := sdk.AuthSession{
		ConsumerID: c.ID,
		ExpireAt:   time.Now().Add(duration),
		GroupIDs:   c.GroupIDs,
		Scopes:     c.Scopes,
	}

	if err := InsertSession(db, &s); err != nil {
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
		ID:       s.ID,
		GroupIDs: s.GroupIDs,
		Scopes:   s.Scopes,
		StandardClaims: jwt.StandardClaims{
			Issuer:    IssuerName,
			Subject:   s.ConsumerID,
			Id:        s.ID,
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: s.ExpireAt.Unix(),
		},
	})
	return SignJWT(jwtToken)
}

// CheckSessionJWT validate given session jwt token.
func CheckSessionJWT(jwtToken string) (*jwt.Token, error) {
	token, err := jwt.ParseWithClaims(jwtToken, &sdk.AuthSessionJWTClaims{}, VerifyJWT)
	if err != nil {
		return nil, sdk.NewErrorWithStack(err, sdk.WithStack(sdk.ErrUnauthorized))
	}

	if _, ok := token.Claims.(*sdk.AuthSessionJWTClaims); ok && token.Valid {
		return token, nil
	}

	return nil, sdk.WithStack(sdk.ErrUnauthorized)
}

// SessionCleaner must be run as a goroutine
func SessionCleaner(c context.Context, dbFunc func() *gorp.DbMap) {
	log.Info("Initializing session cleaner...")
	db := dbFunc()
	tick := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("SessionCleaner> Exiting clean session: %v", c.Err())
				return
			}
		case <-tick.C:
			sessions, err := LoadExpiredSessions(c, db)
			if err != nil {
				log.Error("SessionCleaner> unable to load expired sessions %v", err)
			}
			for _, s := range sessions {
				if err := DeleteSessionByID(db, s.ID); err != nil {
					log.Error("SessionCleaner> unable to delete session %s: %v", s.ID, err)
				}
				log.Debug("SessionCleaner> expired session %s deleted", s.ID)
			}
		}
	}
}
