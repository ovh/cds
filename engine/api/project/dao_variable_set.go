package project

import (
	"context"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
)

func WithVariableSetItems(ctx context.Context, _ *gorpmapper.Mapper, db gorp.SqlExecutor, i interface{}) error {
	switch target := i.(type) {
	case *dbProjectVariableSet:
		items, err := LoadVariableSetAllItem(ctx, db, target.ID)
		if err != nil {
			return err
		}
		target.ProjectVariableSet.Items = items
	default:
		panic(fmt.Sprintf("WithItems: unsupported target %T", i))
	}
	return nil
}

func getVariableSet(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query, opts ...gorpmapper.GetOptionFunc) (*sdk.ProjectVariableSet, error) {
	var dbVarSet dbProjectVariableSet
	found, err := gorpmapping.Get(ctx, db, query, &dbVarSet, opts...)
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	if !found {
		return nil, sdk.NewErrorFrom(sdk.ErrNotFound, "unable to found variable set")
	}
	isValid, err := gorpmapping.CheckSignature(dbVarSet, dbVarSet.Signature)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	if !isValid {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &dbVarSet.ProjectVariableSet, nil
}

func getAllVariableSets(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.ProjectVariableSet, error) {
	var dbVarSets []dbProjectVariableSet
	if err := gorpmapping.GetAll(ctx, db, query, &dbVarSets); err != nil {
		return nil, sdk.WithStack(err)
	}
	varSets := make([]sdk.ProjectVariableSet, 0, len(dbVarSets))
	for _, vs := range dbVarSets {
		isValid, err := gorpmapping.CheckSignature(vs, vs.Signature)
		if err != nil {
			return nil, sdk.WithStack(err)
		}
		if !isValid {
			log.ErrorWithStackTrace(ctx, err)
			continue
		}
		varSets = append(varSets, vs.ProjectVariableSet)
	}
	return varSets, nil
}

func InsertVariableSet(ctx context.Context, db gorpmapper.SqlExecutorWithTx, varSet *sdk.ProjectVariableSet) error {
	varSet.ID = sdk.UUID()
	varSet.Created = time.Now()
	dbVarSet := dbProjectVariableSet{ProjectVariableSet: *varSet}
	if err := gorpmapping.InsertAndSign(ctx, db, &dbVarSet); err != nil {
		return sdk.WithStack(err)
	}
	*varSet = dbVarSet.ProjectVariableSet
	return nil
}

func DeleteVariableSet(ctx context.Context, db gorpmapper.SqlExecutorWithTx, varSet sdk.ProjectVariableSet) error {
	dbVarSet := dbProjectVariableSet{ProjectVariableSet: varSet}
	if err := gorpmapping.Delete(db, &dbVarSet); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

func LoadVariableSetByName(ctx context.Context, db gorp.SqlExecutor, projectKey string, name string, opts ...gorpmapper.GetOptionFunc) (*sdk.ProjectVariableSet, error) {
	query := gorpmapping.NewQuery("SELECT * FROM project_variable_set WHERE project_key = $1 AND name = $2").Args(projectKey, name)
	return getVariableSet(ctx, db, query, opts...)
}

func LoadVariableSetsByProject(ctx context.Context, db gorp.SqlExecutor, projectKey string) ([]sdk.ProjectVariableSet, error) {
	query := gorpmapping.NewQuery("SELECT * FROM project_variable_set WHERE project_key = $1").Args(projectKey)
	return getAllVariableSets(ctx, db, query)
}

func LoadVariableSetItem(ctx context.Context, db gorp.SqlExecutor, variableSetID string, itemName string, opts ...gorpmapping.GetOptionFunc) (*sdk.ProjectVariableSetItem, error) {
	query := gorpmapping.NewQuery("SELECT * FROM project_variable_set_text WHERE project_variable_set_id = $1 AND name = $2").Args(variableSetID, itemName)
	item, err := getVariableSetItemText(ctx, db, query)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return nil, err
	}
	if sdk.ErrorIs(err, sdk.ErrNotFound) {
		query = gorpmapping.NewQuery("SELECT * FROM project_variable_set_secret WHERE project_variable_set_id = $1 AND name = $2").Args(variableSetID, itemName)
		secret, err := getVariableSetItemSecret(ctx, db, query, opts...)
		if err != nil {
			return nil, err
		}
		return secret, nil
	}
	return item, nil
}

func LoadVariableSetItemWithType(ctx context.Context, db gorp.SqlExecutor, variableSetID string, itemName string, itemType string, opts ...gorpmapping.GetOptionFunc) (*sdk.ProjectVariableSetItem, error) {
	switch itemType {
	case sdk.ProjectVariableTypeSecret:
		query := gorpmapping.NewQuery("SELECT * FROM project_variable_set_secret WHERE project_variable_set_id = $1 AND name = $2").Args(variableSetID, itemName)
		secret, err := getVariableSetItemSecret(ctx, db, query, opts...)
		if err != nil {
			return nil, err
		}
		return secret, nil
	default:
		query := gorpmapping.NewQuery("SELECT * FROM project_variable_set_text WHERE project_variable_set_id = $1 AND name = $2").Args(variableSetID, itemName)
		itemText, err := getVariableSetItemText(ctx, db, query)
		if err != nil {
			return nil, err
		}
		return itemText, nil
	}
}

func LoadVariableSetAllItem(ctx context.Context, db gorp.SqlExecutor, variableSetID string, opts ...gorpmapper.GetOptionFunc) ([]sdk.ProjectVariableSetItem, error) {
	items := make([]sdk.ProjectVariableSetItem, 0)
	query := gorpmapping.NewQuery("SELECT * FROM project_variable_set_text WHERE project_variable_set_id = $1").Args(variableSetID)
	itemsText, err := getAllVariableSetItemsText(ctx, db, query)
	if err != nil {
		return nil, err
	}
	querySecret := gorpmapping.NewQuery("SELECT * FROM project_variable_set_secret WHERE project_variable_set_id = $1").Args(variableSetID)
	secrets, err := getAllVariableSetItemsSecret(ctx, db, querySecret, opts...)
	if err != nil {
		return nil, err
	}
	items = append(items, itemsText...)
	items = append(items, secrets...)
	return items, nil
}
