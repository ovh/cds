package user

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// LoadOptionFunc loads data on given authentified users.
type LoadOptionFunc func(context.Context, gorp.SqlExecutor, ...*sdk.AuthentifiedUser) error

// LoadOptions for authentified users.
var LoadOptions = struct {
	WithContacts  LoadOptionFunc
	WithFavorites LoadOptionFunc // TODO
	WithGroups    LoadOptionFunc // TODO
}{
	WithContacts: loadContacts,
}

func loadContacts(ctx context.Context, db gorp.SqlExecutor, aus ...*sdk.AuthentifiedUser) error {
	userIDs := sdk.AuthentifiedUsersToIDs(aus)

	contacts, err := LoadContactsByUserIDs(ctx, db, userIDs)
	if err != nil {
		return err
	}

	mapUsers := make(map[string][]sdk.UserContact, len(contacts))
	for i := range contacts {
		if _, ok := mapUsers[contacts[i].UserID]; !ok {
			mapUsers[contacts[i].UserID] = make([]sdk.UserContact, 0, len(contacts))
		}
		mapUsers[contacts[i].UserID] = append(mapUsers[contacts[i].UserID], contacts[i])
	}

	for i := range aus {
		if _, ok := mapUsers[aus[i].ID]; ok {
			aus[i].Contacts = mapUsers[aus[i].ID]
		}
	}

	return nil
}
