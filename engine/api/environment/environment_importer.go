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
