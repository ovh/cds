package project

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
)

func getVariableSetItemSecret(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query, opts ...gorpmapping.GetOptionFunc) (*sdk.ProjectVariableSetItem, error) {
	var dbVarSetSecret dbProjectVariableSetItemSecret
	found, err := gorpmapping.Get(ctx, db, query, &dbVarSetSecret, opts...)
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	if !found {
		return nil, sdk.NewErrorFrom(sdk.ErrNotFound, "unable to found variable set item")
	}
	isValid, err := gorpmapping.CheckSignature(dbVarSetSecret, dbVarSetSecret.Signature)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	if !isValid {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return dbVarSetSecret.Item(), nil
}

func getAllVariableSetItemsSecret(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query, opts ...gorpmapping.GetOptionFunc) ([]sdk.ProjectVariableSetItem, error) {
	var dbVarSetsSecrets []dbProjectVariableSetItemSecret
	if err := gorpmapping.GetAll(ctx, db, query, &dbVarSetsSecrets, opts...); err != nil {
		return nil, sdk.WithStack(err)
	}
	varSets := make([]sdk.ProjectVariableSetItem, 0, len(dbVarSetsSecrets))
	for _, vs := range dbVarSetsSecrets {
		isValid, err := gorpmapping.CheckSignature(vs, vs.Signature)
		if err != nil {
			return nil, sdk.WithStack(err)
		}
		if !isValid {
			log.ErrorWithStackTrace(ctx, err)
			continue
		}
		varSets = append(varSets, *vs.Item())
	}
	return varSets, nil
}

func InsertVariableSetItemSecret(ctx context.Context, db gorpmapper.SqlExecutorWithTx, varSetSecret *sdk.ProjectVariableSetItem) error {
	varSetSecret.ID = sdk.UUID()
	varSetSecret.LastModified = time.Now()
	dbVarSetSecret := newDbProjectVariableSetItemSecret(*varSetSecret)
	if err := gorpmapping.InsertAndSign(ctx, db, &dbVarSetSecret); err != nil {
		return sdk.WithStack(err)
	}
	*varSetSecret = *dbVarSetSecret.Item()
	return nil
}

func UpdateVariableSetItemSecret(ctx context.Context, db gorpmapper.SqlExecutorWithTx, varSetSecret *sdk.ProjectVariableSetItem) error {
	varSetSecret.LastModified = time.Now()
	dbVarSetSecret := newDbProjectVariableSetItemSecret(*varSetSecret)
	if err := gorpmapping.UpdateAndSign(ctx, db, &dbVarSetSecret); err != nil {
		return sdk.WithStack(err)
	}
	*varSetSecret = *dbVarSetSecret.Item()
	return nil
}

func DeleteVariableSetItemSecret(ctx context.Context, db gorpmapper.SqlExecutorWithTx, varSetSecret sdk.ProjectVariableSetItem) error {
	dbVarSetSecret := newDbProjectVariableSetItemSecret(varSetSecret)
	if err := gorpmapping.Delete(db, &dbVarSetSecret); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}
