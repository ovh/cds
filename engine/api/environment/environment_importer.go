package environment

import (
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//Import import or reuser the provided environment
func Import(db gorp.SqlExecutor, proj *sdk.Project, env *sdk.Environment, msgChan chan<- sdk.Message, u *sdk.User) error {
	exists, err := Exists(db, proj.Key, env.Name)
	if err != nil {
		return err
	}

	//If environment exists, reload it
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
		return sdk.WrapError(err, "Unable to create env %s on project %s(%d) ", env.Name, env.ProjectKey, env.ProjectID)
	}

	//Insert all variables
	for i := range env.Variable {
		if err := InsertVariable(db, env.ID, &env.Variable[i], u); err != nil {
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
			msgChan <- sdk.NewMessage(sdk.MsgEnvironmentKeyCreated, strings.ToUpper(k.Type), k.Name, env.Name)
		}
	}

	if msgChan != nil {
		msgChan <- sdk.NewMessage(sdk.MsgEnvironmentCreated, env.Name)
	}

	return nil
}

//ImportInto import variables and groups on an existing environment
func ImportInto(db gorp.SqlExecutor, proj *sdk.Project, env *sdk.Environment, into *sdk.Environment, msgChan chan<- sdk.Message, u *sdk.User) error {
	var updateVar = func(v *sdk.Variable) {
		log.Debug("ImportInto> Updating var %s", v.Name)

		varBefore, errV := GetVariableByID(db, into.ID, v.ID, WithClearPassword())
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

	var insertVar = func(v *sdk.Variable) {
		log.Debug("ImportInto> Creating var %s", v.Name)
		if err := InsertVariable(db, into.ID, v, u); err != nil {
			msgChan <- sdk.NewMessage(sdk.MsgEnvironmentVariableCannotBeCreated, v.Name, into.Name, err)
			return
		}
		msgChan <- sdk.NewMessage(sdk.MsgEnvironmentVariableCreated, v.Name, into.Name)
	}

	for i := range env.Variable {
		log.Debug("ImportInto> Checking >> %s", env.Variable[i].Name)
		var found bool
		for j := range into.Variable {
			log.Debug("ImportInto> \t with >> %s", into.Variable[j].Name)
			if env.Variable[i].Name == into.Variable[j].Name {
				env.Variable[i].ID = into.Variable[j].ID
				found = true
				updateVar(&env.Variable[i])
				break
			}
		}
		if !found {
			insertVar(&env.Variable[i])
		}
	}

	log.Debug("ImportInto> Done")

	return nil
}
