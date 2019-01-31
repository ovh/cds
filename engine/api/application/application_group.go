package application

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// LoadGroupByApplication loads all the groups on the given application
func LoadGroupByApplication(db gorp.SqlExecutor, app *sdk.Application) error {
	app.ApplicationGroups = []sdk.GroupPermission{}
	query := `SELECT "group".id, "group".name, application_group.role FROM "group"
	 		  JOIN application_group ON application_group.group_id = "group".id
	 		  WHERE application_group.application_id = $1 ORDER BY "group".name ASC`
	rows, errq := db.Query(query, app.ID)
	if errq != nil {
		return errq
	}
	defer rows.Close()

	for rows.Next() {
		var group sdk.Group
		var perm int
		if err := rows.Scan(&group.ID, &group.Name, &perm); err != nil {
			return err
		}
		app.ApplicationGroups = append(app.ApplicationGroups, sdk.GroupPermission{
			Group:      group,
			Permission: perm,
		})
	}
	return nil
}

// LoadPermissions loads all applications where group has access
func LoadPermissions(db gorp.SqlExecutor, groupID int64) ([]sdk.ApplicationGroup, error) {
	res := []sdk.ApplicationGroup{}
	query := `SELECT project.projectKey,
	                 application.name,
	                 application.id,
					 application_group.role, application.last_modified
	      FROM application
	      JOIN application_group ON application_group.application_id = application.id
	 	  JOIN project ON application.project_id = project.id
	 	  WHERE application_group.group_id = $1
	 	  ORDER BY application.name ASC`
	rows, err := db.Query(query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var application sdk.Application
		var perm int
		err = rows.Scan(&application.ProjectKey, &application.Name, &application.ID, &perm, &application.LastModified)
		if err != nil {
			return nil, sdk.WrapError(err, "LoadPermission  (%d)", groupID)
		}
		res = append(res, sdk.ApplicationGroup{
			Application: application,
			Permission:  perm,
		})
	}
	return res, nil
}

// AddGroup Link the given groups and the given application
func AddGroup(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, a *sdk.Application, u *sdk.User, groupPermission ...sdk.GroupPermission) error {
	for i := range groupPermission {
		gp := &groupPermission[i]
		g := &gp.Group
		if g.ID == 0 {
			var errG error
			g, errG = group.LoadGroup(db, g.Name)
			if errG != nil {
				return sdk.WrapError(errG, "AddGroup: Cannot find %s", g.Name)
			}
		}

		log.Debug("application.AddGroup> proj=%s app=%s group=%s", proj.Name, a.Name, g.Name)
		groupAttachedToApp, erra := group.CheckGroupInApplication(db, a.ID, g.ID)
		if erra != nil {
			return sdk.WrapError(erra, "AddGroup> Unable to check group in application %s project:%s", a.Name, proj.Name)
		}

		if !groupAttachedToApp {
			if err := group.InsertGroupInApplication(db, a.ID, g.ID, gp.Permission); err != nil {
				return sdk.WrapError(err, "Unable to inserting in application_group %d %d %d - application %s project:%s", a.ID, g.ID, gp.Permission, a.Name, proj.Name)
			}
		}
		// If the group has only read permission on application, let it go with read permission on projet, pipeline and environment
		// If the group has more than read permission on Application, it will have read & execute permission on projet, pipeline and environment
		perm := permission.PermissionReadExecute
		if gp.Permission == permission.PermissionRead {
			perm = permission.PermissionRead
		}

		//Check association with project
		groupAttachedToProject, errc := group.CheckGroupInProject(db, proj.ID, g.ID)
		if errc != nil {
			return sdk.WrapError(errc, "AddGroup> Unable to check group in project")
		}

		if !groupAttachedToProject {
			if err := group.InsertGroupInProject(db, proj.ID, g.ID, perm); err != nil {
				return sdk.WrapError(err, "Cannot add group %s in project %s", g.Name, proj.Name)
			}
		}
	}
	return nil
}
