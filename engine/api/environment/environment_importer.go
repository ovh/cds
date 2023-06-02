package environment

import (
	"context"
	"strings"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

// Import import or reuser the provided environment
func Import(db gorpmapper.SqlExecutorWithTx, proj sdk.Project, env *sdk.Environment, msgChan chan<- sdk.Message, u sdk.Identifiable) error {
	exists, err := Exists(db, proj.Key, env.Name)
	if err != nil {
		return err
	}

	// If environment exists, reload it
	if exists {
		if msgChan != nil {
			msgChan <- sdk.NewMessage(sdk.MsgEnvironmentExists, env.Name)
		}

		//Reload environment
		e, err := LoadEnvironmentByName(db, proj.Key, env.Name)
		if err != nil {
			return err
		}
		*env = *e

		return nil
	}

	//Else create it
	env.ProjectID = proj.ID
	env.ProjectKey = proj.Key
	if err := InsertEnvironment(db, env); err != nil {
		return sdk.WrapError(err, "unable to create env %q on project %q", env.Name, env.ProjectKey)
	}

	//Insert all variables
	for i := range env.Variables {
		if err := InsertVariable(db, env.ID, &env.Variables[i], u); err != nil {
			return err
		}
	}

	//Insert keys
	for _, k := range env.Keys {
		k.EnvironmentID = env.ID
		if err := InsertKey(db, &k); err != nil {
			return sdk.WrapError(err, "Unable to insert key %s", k.Name)
		}
		if msgChan != nil {
			msgChan <- sdk.NewMessage(sdk.MsgEnvironmentKeyCreated, strings.ToUpper(string(k.Type)), k.Name, env.Name)
		}
	}

	if msgChan != nil {
		msgChan <- sdk.NewMessage(sdk.MsgEnvironmentCreated, env.Name)
	}

	return nil
}

// ImportInto import variables and groups on an existing environment
func ImportInto(ctx context.Context, db gorpmapper.SqlExecutorWithTx, env *sdk.Environment, into *sdk.Environment, msgChan chan<- sdk.Message, u sdk.Identifiable) error {
	//Delete all Variables
	if err := DeleteAllVariables(db, into.ID); err != nil {
		return err
	}

	///Delete all Keys
	if err := DeleteAllEnvironmentKeys(db, into.ID); err != nil {
		return err
	}

	for i := range env.Variables {
		if err := InsertVariable(db, into.ID, &env.Variables[i], u); err != nil {
			return err
		}
	}

	for i := range env.Keys {
		if err := InsertKey(db, &env.Keys[i]); err != nil {
			return err
		}
	}

	if err := UpdateEnvironment(db, env); err != nil {
		return sdk.WrapError(err, "unable to update environment")
	}

	log.Debug(ctx, "ImportInto> Done")

	return nil
}
