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

func getVariableSetItemText(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) (*sdk.ProjectVariableSetItem, error) {
	var dbVarSetText dbProjectVariableSetItemText
	found, err := gorpmapping.Get(ctx, db, query, &dbVarSetText)
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	if !found {
		return nil, sdk.NewErrorFrom(sdk.ErrNotFound, "unable to found variable set item")
	}
	isValid, err := gorpmapping.CheckSignature(dbVarSetText, dbVarSetText.Signature)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	if !isValid {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return dbVarSetText.Item(), nil
}

func getAllVariableSetItemsText(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.ProjectVariableSetItem, error) {
	var dbVarSetsTexts []dbProjectVariableSetItemText
	if err := gorpmapping.GetAll(ctx, db, query, &dbVarSetsTexts); err != nil {
		return nil, sdk.WithStack(err)
	}
	varSets := make([]sdk.ProjectVariableSetItem, 0, len(dbVarSetsTexts))
	for _, vs := range dbVarSetsTexts {
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

func InsertVariableSetItemText(ctx context.Context, db gorpmapper.SqlExecutorWithTx, varSetText *sdk.ProjectVariableSetItem) error {
	varSetText.ID = sdk.UUID()
	varSetText.LastModified = time.Now()
	dbVarSetText := newDbProjectVariableSetItemText(*varSetText)
	if err := gorpmapping.InsertAndSign(ctx, db, &dbVarSetText); err != nil {
		return sdk.WithStack(err)
	}
	*varSetText = *dbVarSetText.Item()
	return nil
}

func UpdateVariableSetItemText(ctx context.Context, db gorpmapper.SqlExecutorWithTx, varSetText *sdk.ProjectVariableSetItem) error {
	varSetText.LastModified = time.Now()
	dbVarSetText := newDbProjectVariableSetItemText(*varSetText)
	if err := gorpmapping.UpdateAndSign(ctx, db, &dbVarSetText); err != nil {
		return sdk.WithStack(err)
	}
	*varSetText = *dbVarSetText.Item()
	return nil
}

func DeleteVariableSetItemText(ctx context.Context, db gorpmapper.SqlExecutorWithTx, varSetText sdk.ProjectVariableSetItem) error {
	dbVarSetText := newDbProjectVariableSetItemText(varSetText)
	if err := gorpmapping.Delete(db, &dbVarSetText); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}
