package environment

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/msg"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

//Import import or reuser the provided environment
func Import(db gorp.SqlExecutor, proj *sdk.Project, env *sdk.Environment, msgChan chan<- msg.Message) error {
	exists, err := Exists(db, proj.Key, env.Name)
	if err != nil {
		return err
	}

	//If environment exists, reload it
	if exists {
		if msgChan != nil {
			msgChan <- msg.New(msg.EnvironmentExists, env.Name)
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
		log.Warning("environment.Exists> Unable to create env %s on project %s(%d) : %s", env.Name, env.ProjectKey, env.ProjectID, err)
		return err
	}

	//If no GroupPermission provided, inherit from project
	if env.EnvironmentGroups == nil {
		env.EnvironmentGroups = proj.ProjectGroups
	}
	if err := group.InsertGroupsInEnvironment(db, env.EnvironmentGroups, env.ID); err != nil {
		log.Warning("environment.Import> unable to import groups in environment %s, %s", env.Name, err)
		return err
	}

	//Insert all variables
	for i := range env.Variable {
		if err := InsertVariable(db, env.ID, &env.Variable[i]); err != nil {
			return err
		}
	}

	if msgChan != nil {
		msgChan <- msg.New(msg.EnvironmentCreated, env.Name)
	}

	return nil
}

//ImportInto import variables and groups on an existing environment
func ImportInto(db gorp.SqlExecutor, proj *sdk.Project, env *sdk.Environment, into *sdk.Environment, msgChan chan<- msg.Message) error {

	var updateVar = func(v *sdk.Variable) {
		log.Debug("ImportInto> Updating var %s", v.Name)
		if err := UpdateVariable(db, into.ID, v); err != nil {
			msgChan <- msg.New(msg.VariableCannotBeUpdated, v.Name, into.Name, err)
			return
		}
		msgChan <- msg.New(msg.VariableUpdated, v.Name, into.Name)
	}

	var insertVar = func(v *sdk.Variable) {
		log.Debug("ImportInto> Creating var %s", v.Name)
		if err := InsertVariable(db, into.ID, v); err != nil {
			msgChan <- msg.New(msg.VariableCannotBeCreated, v.Name, into.Name, err)
			return
		}
		msgChan <- msg.New(msg.VariableCreated, v.Name, into.Name)
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

	return nil
}
