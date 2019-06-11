package accesstoken

import (
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// NewSession returns a new session for a given auth consumer.
func NewSession(db gorp.SqlExecutor, c *sdk.AuthConsumer, expiration time.Time) (*sdk.AuthSession, error) {
	s := sdk.AuthSession{
		ID:         sdk.UUID(),
		ConsumerID: c.ID,
		ExpireAt:   expiration,
		Created:    time.Now(),
		GroupIDs:   c.GroupIDs,
		Scopes:     c.Scopes,
	}

	if err := InsertSession(db, &s); err != nil {
		return nil, err
	}

	return &s, nil
}
