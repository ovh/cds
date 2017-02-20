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
				return sdk.ErrGroupNotFound
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
				log.Warning("AddGroup> Unable to inserting in application_group %d %d %d: s%", a.ID, g.ID, gp.Permission, err)
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
