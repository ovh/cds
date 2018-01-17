package environment

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/group"
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
		return sdk.WrapError(err, "environment.Exists> Unable to create env %s on project %s(%d) ", env.Name, env.ProjectKey, env.ProjectID)
	}

	//If no GroupPermission provided, inherit from project
	if len(env.EnvironmentGroups) == 0 {
		env.EnvironmentGroups = proj.ProjectGroups
	}
	if err := group.InsertGroupsInEnvironment(db, env.EnvironmentGroups, env.ID); err != nil {
		return sdk.WrapError(err, "environment.Import> unable to import groups in environment %s ", env.Name)
	}

	//Insert all variables
	for i := range env.Variable {
		if err := InsertVariable(db, env.ID, &env.Variable[i], u); err != nil {
			return err
		}
	}

	if msgChan != nil {
		msgChan <- sdk.NewMessage(sdk.MsgEnvironmentCreated, env.Name)
	}

	return nil
}

//ImportInto import variables and groups on an existing environment
func ImportInto(db gorp.SqlExecutor, proj *sdk.Project, env *sdk.Environment, into *sdk.Environment, msgChan chan<- sdk.Message, u *sdk.User) error {

	if len(into.EnvironmentGroups) == 0 {
		if err := loadGroupByEnvironment(db, into); err != nil {
			return err
		}
	}

	var updateVar = func(v *sdk.Variable) {
		log.Debug("ImportInto> Updating var %s", v.Name)
		if err := UpdateVariable(db, into.ID, v, u); err != nil {
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

	var updateGroupInEnv = func(groupName string, groupID int64, role int) {
		log.Debug("ImportInto> Updating group %s", groupID)
		if err := group.UpdateGroupRoleInEnvironment(db, into.ID, groupID, role); err != nil {
			msgChan <- sdk.NewMessage(sdk.MsgEnvironmentGroupCannotBeUpdated, groupName, into.Name, err)
			return
		}
		msgChan <- sdk.NewMessage(sdk.MsgEnvironmentGroupUpdated, groupName, into.Name)
	}

	var insertGroupInEnv = func(groupName string, role int) {
		log.Debug("ImportInto> Adding group %s", groupName)
		g, err := group.LoadGroup(db, groupName)
		if err != nil {
			msgChan <- sdk.NewMessage(sdk.MsgEnvironmentGroupCannotBeCreated, groupName, into.Name, err)
			return
		}

		if err := group.InsertGroupInEnvironment(db, into.ID, g.ID, role); err != nil {
			msgChan <- sdk.NewMessage(sdk.MsgEnvironmentGroupCannotBeCreated, groupName, into.Name, err)
			return
		}
		msgChan <- sdk.NewMessage(sdk.MsgEnvironmentGroupCreated, groupName, into.Name)
	}

	var deleteGroupInEnv = func(groupName string, groupID int64) {
		if err := group.DeleteGroupFromEnvironment(db, into.ID, groupID); err != nil {
			msgChan <- sdk.NewMessage(sdk.MsgEnvironmentGroupCannotBeDeleted, groupName, into.Name, err)
			return
		}
		msgChan <- sdk.NewMessage(sdk.MsgEnvironmentGroupDeleted, groupName, into.Name)
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

	for i := range env.EnvironmentGroups {
		log.Debug("ImportInto> Checking >> %s", env.EnvironmentGroups[i].Group.Name)
		var found bool
		for j := range into.EnvironmentGroups {
			log.Debug("ImportInto> \t with >> %s", into.EnvironmentGroups[j].Group.Name)
			if env.EnvironmentGroups[i].Group.Name == into.EnvironmentGroups[j].Group.Name {
				env.EnvironmentGroups[i].Group.ID = into.EnvironmentGroups[j].Group.ID
				found = true
				updateGroupInEnv(env.EnvironmentGroups[i].Group.Name, env.EnvironmentGroups[i].Group.ID, env.EnvironmentGroups[i].Permission)
				break
			}
		}
		if !found {
			insertGroupInEnv(env.EnvironmentGroups[i].Group.Name, env.EnvironmentGroups[i].Permission)
		}
	}

	for i := range into.EnvironmentGroups {
		log.Debug("ImportInto>Delete ?> Checking >> %s", into.EnvironmentGroups[i].Group.Name)

		var found bool
		for j := range env.EnvironmentGroups {
			log.Debug("ImportInto>Delete ?> \t with >> %s", env.EnvironmentGroups[j].Group.Name)
			if into.EnvironmentGroups[i].Group.Name == env.EnvironmentGroups[j].Group.Name {
				found = true
			}
		}
		if !found {
			log.Debug("ImportInto>Delete ?> \t with >> %s", into.EnvironmentGroups[i].Group.Name)
			deleteGroupInEnv(into.EnvironmentGroups[i].Group.Name, into.EnvironmentGroups[i].Group.ID)
		}
	}

	log.Debug("ImportInto> Done")

	return nil
}
