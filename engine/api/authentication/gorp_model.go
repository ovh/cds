package authentication

import (
	"fmt"
	"time"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

type authConsumer struct {
	sdk.AuthConsumer
	gorpmapping.SignedEntity
}

func (c authConsumer) Canonical() ([]byte, error) {
	var canonical string
	canonical += c.ID
	if c.ParentID != nil {
		canonical += *c.ParentID
	}
	canonical += c.AuthentifiedUserID
	canonical += string(c.Type)
	canonical += fmt.Sprintf("%v", c.Data)
	canonical += c.Created.In(time.UTC).Format(time.RFC3339)
	canonical += fmt.Sprintf("%v", c.GroupIDs)
	canonical += fmt.Sprintf("%v", c.Scopes)
	return []byte(canonical), nil
}

type authSession struct {
	sdk.AuthSession
	gorpmapping.SignedEntity
}

func (s authSession) Canonical() ([]byte, error) {
	var canonical string
	canonical += s.ID
	canonical += s.ConsumerID
	canonical += s.ExpireAt.In(time.UTC).Format(time.RFC3339)
	canonical += s.Created.In(time.UTC).Format(time.RFC3339)
	canonical += fmt.Sprintf("%v", s.GroupIDs)
	canonical += fmt.Sprintf("%v", s.Scopes)
	return []byte(canonical), nil
}

func init() {
	gorpmapping.Register(
		gorpmapping.New(authConsumer{}, "auth_consumer", false, "id"),
		gorpmapping.New(authSession{}, "auth_session", false, "id"),
	)
}
