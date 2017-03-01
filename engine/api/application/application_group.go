package application

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// LoadGroupByApplication loads all the groups on the given application
func LoadGroupByApplication(db gorp.SqlExecutor, app *sdk.Application) error {
	app.ApplicationGroups = []sdk.GroupPermission{}
	query := `SELECT "group".id, "group".name, application_group.role FROM "group"
	 		  JOIN application_group ON application_group.group_id = "group".id
	 		  WHERE application_group.application_id = $1 ORDER BY "group".name ASC`
	rows, err := db.Query(query, app.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var group sdk.Group
		var perm int
		err = rows.Scan(&group.ID, &group.Name, &perm)
		if err != nil {
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
func LoadPermissions(db gorp.SqlExecutor, group *sdk.Group) error {
	query := `
		  SELECT project.projectKey,
	                 application.name,
	                 application.id,
					 application_group.role
	      FROM application
	      JOIN application_group ON application_group.application_id = application.id
	 	  JOIN project ON application.project_id = project.id
	 	  WHERE application_group.group_id = $1
	 	  ORDER BY application.name ASC`
	rows, err := db.Query(query, group.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var application sdk.Application
		var perm int
		err = rows.Scan(&application.ProjectKey, &application.Name, &application.ID, &perm)
		if err != nil {
			return sdk.WrapError(err, "LoadPermission %s (%d)", group.Name, group.ID)
		}
		group.ApplicationGroups = append(group.ApplicationGroups, sdk.ApplicationGroup{
			Application: application,
			Permission:  perm,
		})
	}
	return nil
}

// AddGroup Link the given groups and the given application
func AddGroup(db gorp.SqlExecutor, proj *sdk.Project, a *sdk.Application, groupPermission ...sdk.GroupPermission) error {
	for i := range groupPermission {
		gp := &groupPermission[i]
		g := &gp.Group
		if g.ID == 0 {
			var errG error
			g, errG = group.LoadGroup(db, g.Name)
			if errG != nil {
				log.Warning("AddGroup: Cannot find %s: %s\n", g.Name, errG)
				return errG
			}
		}

		log.Debug("application.AddGroup> proj=%s app=%s group=%s", proj.Name, a.Name, g.Name)
		groupAttachedToApp, erra := group.CheckGroupInApplication(db, a.ID, g.ID)
		if erra != nil {
			log.Warning("AddGroup> Unable to check group in application %s", erra)
			return erra
		}

		if !groupAttachedToApp {
			if err := group.InsertGroupInApplication(db, a.ID, g.ID, gp.Permission); err != nil {
				log.Warning("AddGroup> Unable to inserting in application_group %d %d %d: %s", a.ID, g.ID, gp.Permission, err)
				return err
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
			log.Warning("AddGroup> Unable to check group in project %s", errc)
			return errc
		}

		if !groupAttachedToProject {
			if err := group.InsertGroupInProject(db, proj.ID, g.ID, perm); err != nil {
				log.Warning("AddGroup> Cannot add group %s in project %s:  %s\n", g.Name, proj.Name, err)
				return err
			}
		}

		//For all attached pipelines
		for _, p := range a.Pipelines {
			//Check association with pipeline
			log.Debug("application.AddGroup> proj=%s pip=%d group=%s", proj.Name, p.Pipeline.ID, g.Name)
			groupAttachedToPipeline, errp := group.CheckGroupInPipeline(db, p.Pipeline.ID, g.ID)
			if errp != nil {
				log.Warning("AddGroup> Unable to check group in pipeline %s", errp)
				return errp
			}
			if !groupAttachedToPipeline {
				if err := group.InsertGroupInPipeline(db, p.Pipeline.ID, g.ID, perm); err != nil {
					log.Warning("AddGroup> Cannot add group %s in pipeline %s:  %s\n", g.Name, p.Pipeline.Name, err)
					return err
				}

				if err := pipeline.UpdateLastModified(db, p.Pipeline.ID); err != nil {
					log.Warning("AddGroup> Cannot update pipeline %s:  %s\n", p.Pipeline.Name, err)
					return err
				}

			}

			//Check environments
			for _, t := range p.Triggers {
				if t.DestApplication.ID == a.ID {
					groupAttachedToEnv, erre := group.IsInEnvironment(db, t.DestEnvironment.ID, g.ID)
					if erre != nil {
						log.Warning("AddGroup> Unable to check group in env %s", erre)
						return erre
					}

					if !groupAttachedToEnv {
						if err := group.InsertGroupInEnvironment(db, t.DestEnvironment.ID, g.ID, perm); err != nil {
							log.Warning("AddGroup> Cannot add group %s in env %s:  %s\n", g.Name, t.DestEnvironment.Name, err)
							return err
						}

						if err := environment.UpdateLastModified(db, t.DestEnvironment.ID); err != nil {
							log.Warning("AddGroup> Cannot update env %s:  %s\n", t.DestEnvironment.Name, err)
							return err
						}
					}
				}
				if t.SrcApplication.ID == a.ID {
					groupAttachedToEnv, erre := group.IsInEnvironment(db, t.SrcEnvironment.ID, g.ID)
					if erre != nil {
						log.Warning("AddGroup> Unable to check group in env %s", erre)
						return erre
					}

					if !groupAttachedToEnv {
						if err := group.InsertGroupInEnvironment(db, t.SrcEnvironment.ID, g.ID, perm); err != nil {
							log.Warning("AddGroup> Cannot add group %s in env %s:  %s\n", g.Name, t.SrcEnvironment.Name, err)
							return err
						}

						if err := environment.UpdateLastModified(db, t.SrcEnvironment.ID); err != nil {
							log.Warning("AddGroup> Cannot update env %s:  %s\n", t.SrcEnvironment.Name, err)
							return err
						}
					}
				}
			}
		}
	}
	return nil
}
