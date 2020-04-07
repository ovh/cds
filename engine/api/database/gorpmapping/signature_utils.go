package gorpmapping

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

func ListSignedEntities() []string {
	var signedEntities []string
	for k, v := range Mapping {
		if v.SignedEntity {
			signedEntities = append(signedEntities, k)
		}
	}
	return signedEntities
}

func ListCanonicalFormsByEntity(db gorp.SqlExecutor, entity string) ([]sdk.CanonicalFormUsage, error) {
	e, ok := Mapping[entity]
	if !ok {
		return nil, sdk.WithStack(errors.New("unknown entity"))
	}
	if !e.SignedEntity {
		return nil, sdk.WithStack(errors.New("entity is not signed"))
	}

	var res []sdk.CanonicalFormUsage
	if _, err := db.Select(&res, fmt.Sprintf(`select signer, count(sig) as number from "%s" group by signer`, e.Name)); err != nil {
		return nil, sdk.WithStack(err)
	}

	x := e.Target.(Canonicaller)
	lastestCanonicalForm, _ := x.Canonical().Latest()
	sha := getSigner(lastestCanonicalForm)

	for i := range res {
		if res[i].Signer == sha {
			res[i].Latest = true
		}
	}

	return res, nil
}

func ListTuplesByEntity(db gorp.SqlExecutor, entity string) ([]string, error) {
	e, ok := Mapping[entity]
	if !ok {
		return nil, sdk.WithStack(errors.New("unknown entity"))
	}

	var res []string
	if _, err := db.Select(&res, fmt.Sprintf(`select %s::text from "%s"`, e.Keys[0], e.Name)); err != nil {
		return nil, sdk.WithStack(err)
	}

	return res, nil
}

func ListTupleByCanonicalForm(db gorp.SqlExecutor, entity, signer string) ([]string, error) {
	e, ok := Mapping[entity]
	if !ok {
		return nil, sdk.WithStack(errors.New("unknown entity"))
	}
	if !e.SignedEntity {
		return nil, sdk.WithStack(errors.New("entity is not signed"))
	}

	query := fmt.Sprintf(`select %s::text from "%s" where signer = $1`, e.Keys[0], e.Name)
	var res []string
	if _, err := db.Select(&res, query, signer); err != nil {
		return nil, sdk.WithStack(err)
	}

	return res, nil
}

func RollSignedTupleByPrimaryKey(ctx context.Context, db gorp.SqlExecutor, entity string, pk interface{}) error {
	e, ok := Mapping[entity]
	if !ok {
		return errors.New("unknown entity")
	}

	if !e.SignedEntity {
		return errors.New("entity is not signed")
	}

	tuple, err := LoadTupleByPrimaryKey(db, entity, pk)
	if err != nil {
		return err
	}

	if err := UpdateAndSign(ctx, db, tuple.(Canonicaller)); err != nil {
		return err
	}

	return nil
}
