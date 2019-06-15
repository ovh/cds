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
	verifiedUserContacts := make([]*sdk.UserContact, 0, len(cs))
	for i := range cs {
		isValid, err := gorpmapping.CheckSignature(cs[i], cs[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error("user.getAll> user contact %d data corrupted", cs[i].ID)
			continue
		}
		verifiedUserContacts = append(verifiedUserContacts, &cs[i].UserContact)
	}

	ucs := make([]sdk.UserContact, len(verifiedUserContacts))
	for i := range verifiedUserContacts {
		ucs[i] = *verifiedUserContacts[i]
	}

	return ucs, nil
}
