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
	var updateVar = func(v *sdk.EnvironmentVariable) {
		log.Debug(ctx, "ImportInto> Updating var %q with value %q", v.Name, v.Value)

		varBefore, errV := LoadVariable(db, into.ID, v.Name)
		if errV != nil {
			msgChan <- sdk.NewMessage(sdk.MsgEnvironmentVariableCannotBeUpdated, v.Name, into.Name, errV)
			return
		}

		if err := UpdateVariable(db, into.ID, v, varBefore, u); err != nil {
			msgChan <- sdk.NewMessage(sdk.MsgEnvironmentVariableCannotBeUpdated, v.Name, into.Name, err)
			return
		}
		msgChan <- sdk.NewMessage(sdk.MsgEnvironmentVariableUpdated, v.Name, into.Name)
	}

	var insertVar = func(v *sdk.EnvironmentVariable) {
		log.Debug(ctx, "ImportInto> Creating var %s", v.Name)
		if err := InsertVariable(db, into.ID, v, u); err != nil {
			msgChan <- sdk.NewMessage(sdk.MsgEnvironmentVariableCannotBeCreated, v.Name, into.Name, err)
			return
		}
		msgChan <- sdk.NewMessage(sdk.MsgEnvironmentVariableCreated, v.Name, into.Name)
	}

	for i := range env.Variables {
		log.Debug(ctx, "ImportInto> Checking >> %s", env.Variables[i].Name)
		var found bool
		for j := range into.Variables {
			log.Debug(ctx, "ImportInto> \t with >> %s", into.Variables[j].Name)
			if env.Variables[i].Name == into.Variables[j].Name {
				env.Variables[i].ID = into.Variables[j].ID
				found = true
				updateVar(&env.Variables[i])
				break
			}
		}
		if !found {
			insertVar(&env.Variables[i])
		}
	}

	if err := UpdateEnvironment(db, env); err != nil {
		return sdk.WrapError(err, "unable to update environment")
	}

	log.Debug(ctx, "ImportInto> Done")

	return nil
}
