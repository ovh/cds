package user

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getContacts(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]sdk.UserContact, error) {
	cs := []userContact{}

	if err := gorpmapping.GetAll(ctx, db, q, &cs); err != nil {
		return nil, sdk.WrapError(err, "cannot get user contacts")
	}

	// Check signature of data, if invalid do not return it
	verifiedUserContacts := make([]sdk.UserContact, 0, len(cs))
	for i := range cs {
		isValid, err := gorpmapping.CheckSignature(cs[i], cs[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error("user.getContacts> user contact %d data corrupted", cs[i].ID)
			continue
		}
		verifiedUserContacts = append(verifiedUserContacts, cs[i].UserContact)
	}

	return verifiedUserContacts, nil
}

func getContact(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) (*sdk.UserContact, error) {
	var uc userContact

	found, err := gorpmapping.Get(ctx, db, q, &uc)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get user contact")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	isValid, err := gorpmapping.CheckSignature(uc, uc.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error("user.getContact> user contact %d (for user %s) data corrupted", uc.ID, uc.UserID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	return &uc.UserContact, nil
}

// LoadContactsByUserID returns all contacts from database for given user ids.
func LoadContactsByUserID(ctx context.Context, db gorp.SqlExecutor, userIDs []string) ([]sdk.UserContact, error) {
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM user_contact
		WHERE user_id = ANY(string_to_array($1, ',')::text[])
		ORDER BY id ASC
	`).Args(gorpmapping.IDStringsToQueryString(userIDs))
	return getContacts(ctx, db, query)
}

// LoadContactsByTypeAndValue returns a contact for given type and value.
func LoadContactsByTypeAndValue(ctx context.Context, db gorp.SqlExecutor, contactType, value string) (*sdk.UserContact, error) {
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM user_contact
		WHERE type = $1 AND value = $2
	`).Args(contactType, value)
	return getContact(ctx, db, query)
}

// InsertContact in database.
func InsertContact(db gorp.SqlExecutor, c *sdk.UserContact) error {
	dbc := userContact{UserContact: *c}
	if err := gorpmapping.InsertAndSign(db, &dbc); err != nil {
		return err
	}
	*c = dbc.UserContact
	return nil
}

// UpdateContact in database.
func UpdateContact(db gorp.SqlExecutor, c *sdk.UserContact) error {
	dbc := userContact{UserContact: *c}
	if err := gorpmapping.UpdatetAndSign(db, &dbc); err != nil {
		return err
	}
	*c = dbc.UserContact
	return nil
}
