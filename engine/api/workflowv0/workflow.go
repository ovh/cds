package workflowv0

import (
	"context"
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/hook"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/poller"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/scheduler"
	"github.com/ovh/cds/sdk"
)

// GetWorkflowStatus Get workflow updated builds status + scheduler and poller executions
func GetWorkflowStatus(db gorp.SqlExecutor, store cache.Store, projectkey, appName string, user *sdk.User, branchName, remote string, version int64) ([]sdk.PipelineBuild, []sdk.PipelineScheduler, []sdk.RepositoryPoller, []sdk.Hook, error) {
	cdPipelines, err := LoadCDTree(db, store, projectkey, appName, user, branchName, remote, version)
	if err != nil {
		return nil, nil, nil, nil, sdk.WrapError(err, "GetWorkflowStatus> Cannot load workflow")
	}

	var pbs []sdk.PipelineBuild
	var schedulers []sdk.PipelineScheduler
	var pollers []sdk.RepositoryPoller
	var hooks []sdk.Hook

	for _, cdp := range cdPipelines {
		getWorkflowStatus(&pbs, &schedulers, &pollers, &hooks, cdp)
	}

	return pbs, schedulers, pollers, hooks, nil
}

func getWorkflowStatus(pbs *[]sdk.PipelineBuild, schedulers *[]sdk.PipelineScheduler, pollers *[]sdk.RepositoryPoller, hooks *[]sdk.Hook, cdPip sdk.CDPipeline) {
	if len(cdPip.Application.PipelinesBuild) > 0 {
		*pbs = append(*pbs, cdPip.Application.PipelinesBuild...)
	} else if cdPip.Pipeline.LastPipelineBuild != nil {
		*pbs = append(*pbs, *cdPip.Pipeline.LastPipelineBuild)
	}

	if cdPip.Schedulers != nil {
		*schedulers = append(*schedulers, cdPip.Schedulers...)
	}
	if cdPip.Poller != nil {
		*pollers = append(*pollers, *cdPip.Poller)
	}
	if cdPip.Hooks != nil {
		*hooks = append(*hooks, cdPip.Hooks...)
	}
	for _, sub := range cdPip.SubPipelines {
		getWorkflowStatus(pbs, schedulers, pollers, hooks, sub)
	}
}

// LoadCDTree Load the continuous delivery pipeline tree for the given application
func LoadCDTree(db gorp.SqlExecutor, store cache.Store, projectkey, appName string, user *sdk.User, branchName, remote string, version int64) ([]sdk.CDPipeline, error) {
	cdTrees := []sdk.CDPipeline{}

	// Select root trigger element + non triggered pipeline
	query := `
		SELECT  projID, projName,
			appID, appName, appRepoName,
			pipID, pipName, pipType,
			envID, envName,
			hasHook, hasScheduler, hasPoller, hasChild
		FROM
		      -- SELECT FROM scheduler
		      (
			SELECT project.id as projID, project.name as projName,
			     s.application_id as appID, application.name as appName, application.repo_fullname as appRepoName,
			     s.pipeline_id as pipID, pipeline.name as pipName, pipeline.type as pipType,
			     s.environment_id as envID, environment.name as envName,
			     false as hasHook, true as hasScheduler, false as hasPoller, true as hasChild
			FROM pipeline_scheduler s
			JOIN application ON s.application_id = application.id
			JOIN pipeline ON s.pipeline_id = pipeline.id
			JOIN project ON project.id = application.project_id
			JOIN environment ON environment.id = s.environment_id
			WHERE application.name = $2 and project.projectkey = $1
			-- AND NOT TRIGGERED
			AND s.pipeline_id || '-' || s.environment_id  NOT IN (
				SELECT pipID || '-' || envID
				FROM
				(
					SELECT src_pipeline_id as pipID, src_environment_id as envID
					FROM pipeline_trigger
					WHERE src_pipeline_id = s.pipeline_id AND src_environment_id = s.environment_id AND src_application_id = s.application_id
				UNION
					SELECT dest_pipeline_id as pipID, dest_environment_id as envID
					FROM pipeline_trigger
					WHERE dest_pipeline_id = s.pipeline_id AND dest_environment_id = s.environment_id AND src_application_id = s.application_id
				) a
			)
		     ) withScheduler

		     UNION
		     -- SELECT FROM HOOK
		     (
			SELECT project.id as projID, project.name as projName,
			     h.application_id as appID, application.name as appName, application.repo_fullname as appRepoName,
			     h.pipeline_id as pipID, pipeline.name as pipName, pipeline.type as pipType,
			     environment.id as envID, environment.name as envName,
			     true as hasHook, false as hasScheduler, false as hasPoller, true as hasChild
			FROM hook h
			JOIN application ON h.application_id = application.id
			JOIN pipeline ON h.pipeline_id = pipeline.id
			JOIN project ON project.id = application.project_id
			JOIN environment ON environment.id = 1
			WHERE application.name = $2 and project.projectkey = $1
			-- AND NOT TRIGGERED
			AND h.pipeline_id || '-' || environment.id  NOT IN (
				SELECT pipID || '-' || envID
				FROM
				(
					SELECT src_pipeline_id as pipID, src_environment_id as envID
					FROM pipeline_trigger
					WHERE src_pipeline_id = h.pipeline_id AND src_environment_id = environment.id AND src_application_id = h.application_id
				UNION
					SELECT dest_pipeline_id as pipID, dest_environment_id as envID
					FROM pipeline_trigger
					WHERE dest_pipeline_id = h.pipeline_id AND dest_environment_id = environment.id AND src_application_id = h.application_id
				) a
			)
		     )
		     UNION
		     -- SELECT FROM POLLER
		     (
			SELECT project.id as projID, project.name as projName,
			     p.application_id as appID, application.name as appName, application.repo_fullname as appRepoName,
			     p.pipeline_id as pipID, pipeline.name as pipName, pipeline.type as pipType,
			     environment.id as envID, environment.name as envName,
			     false as hasHook, false as hasScheduler, true as hasPoller, true as hasChild
			FROM poller p
			JOIN application ON p.application_id = application.id
			JOIN pipeline ON p.pipeline_id = pipeline.id
			JOIN project ON project.id = application.project_id
			JOIN environment ON environment.id = 1
			WHERE application.name = $2 and project.projectkey = $1
			-- AND NOT TRIGGERED
			AND p.pipeline_id || '-' || environment.id  NOT IN (
				SELECT pipID || '-' || envID
				FROM
				(
					SELECT src_pipeline_id as pipID, src_environment_id as envID
					FROM pipeline_trigger
					WHERE src_pipeline_id = p.pipeline_id AND src_environment_id = environment.id AND src_application_id = p.application_id
				UNION
					SELECT dest_pipeline_id as pipID, dest_environment_id as envID
					FROM pipeline_trigger
					WHERE dest_pipeline_id = p.pipeline_id AND dest_environment_id = environment.id AND src_application_id = p.application_id
				) a
			)
		     )
		     UNION
		     -- ROOT PIPELINE WITH NO TRIGGER
		     (
			SELECT project.id as projID, project.name as projName,
			       application.id as appID, application.name as appName, application.repo_fullname as appRepoName,
			       pipeline.id as pipID, pipeline.name as pipName, pipeline.type as pipType,
			       environment.id as envID, environment.name as EnvName,
			       CASE WHEN COALESCE (count(h.id), 0)>0 THEN true ELSE false END as hasHook,
			       CASE WHEN COALESCE (count(sc.id), 0)>0 THEN true ELSE false END as hasScheduler,
			       CASE WHEN COALESCE (count(p.application_id), 0)>0 THEN true ELSE false END as hasPoller,
			       false as hasChild
			FROM pipeline
			JOIN application_pipeline ON application_pipeline.pipeline_id = pipeline.id
			JOIN application ON application.id = application_pipeline.application_id
			JOIN project ON project.id = application.project_id
			JOIN environment ON environment.id = 1
			LEFT JOIN pipeline_scheduler sc ON sc.application_id = application.id AND sc.pipeline_id = pipeline.id AND sc.environment_id = 1
			LEFT JOIN hook h ON h.application_id = application.id AND h.pipeline_id = pipeline.id
			LEFT JOIN poller p ON p.application_id = application.id AND p.pipeline_id = pipeline.id
			WHERE application.name = $2 and project.projectkey = $1 AND pipeline.id NOT IN (
				select src_pipeline_id
				FROM pipeline_trigger
				WHERE src_application_id = application.id

				UNION

				select dest_pipeline_id
				FROM pipeline_trigger
				WHERE dest_application_id = application.id
				AND src_application_id = application.id
			)
			GROUP by project.id, application.id, pipeline.id, environment.id
		     )
		     UNION
		     (
		     -- ROOT PIPELINE WITH TRIGGER
			SELECT projID, projName,
			       appID, appName, appRepoName,
			       pipID, pipName, pipType,
			       envID, envName,
			       CASE WHEN COALESCE (count(h.id), 0)>0 THEN true ELSE false END as hasHook,
			       CASE WHEN COALESCE (count(sc.id), 0)>0 THEN true ELSE false END as hasScheduler,
			       CASE WHEN COALESCE (count(p.application_id), 0)>0 THEN true ELSE false END as hasPoller,
			       true as hasChild
			FROM (
				-- SELECT ALL SRC APP/PIP/ENV
				SELECT
					distinct on (src_pipeline_id, src_environment_id) src_pipeline_id as pipID, pipeline.name as pipName, pipeline.type as pipType,
					application.project_id as projID, project.name as projName,
					application.id as appID, application.name as appName, application.repo_fullname as appRepoName,
					COALESCE(src_environment_id,1) as envID, environment.name as envName,
					(src_pipeline_id || '-' || COALESCE(src_environment_id,1) ) as dis
				FROM pipeline_trigger
				JOIN application ON src_application_id = application.id
				JOIN project ON project.id = application.project_id
				JOIN pipeline ON pipeline.id = src_pipeline_id
				JOIN environment ON environment.id = src_environment_id
				WHERE src_application_id = (
					SELECT application.id from application
					JOIN project ON project.id = application.project_id
					WHERE project.projectkey = $1 AND application.name = $2
				)
			) roots
			LEFT JOIN pipeline_scheduler sc ON sc.application_id = appID AND sc.pipeline_id = pipID AND sc.environment_id = envID
			LEFT JOIN hook h ON h.application_id = appID AND h.pipeline_id = pipID
			LEFT JOIN poller p ON p.application_id = appID AND p.pipeline_id = pipID
			WHERE (
				dis not in (
					select
						dest_pipeline_id || '-' || COALESCE(dest_environment_id,1)
					FROM pipeline_trigger
					WHERE
						dest_application_id = appID
						AND src_application_id = appID

				)
			)
			GROUP BY projID, projName, appID, appName, appRepoName, pipID, pipName, pipType, envID, envName
		)
		order by
		appID, pipID, envID`

	rows, err := db.Query(query, projectkey, appName)
	if err != nil {
		return nil, sdk.WrapError(err, "LoadCDTree> Cannot load root elements")
	}
	defer rows.Close()

	for rows.Next() {
		var root sdk.CDPipeline
		var typePipeline string
		var hasHook, hasScheduler, hasPoller, hasChild bool

		if err = rows.Scan(&root.Project.ID, &root.Project.Name,
			&root.Application.ID, &root.Application.Name, &root.Application.RepositoryFullname,
			&root.Pipeline.ID, &root.Pipeline.Name, &typePipeline,
			&root.Environment.ID, &root.Environment.Name,
			&hasHook, &hasScheduler, &hasPoller, &hasChild); err != nil {
			return nil, sdk.WrapError(err, "LoadCDTree> Cannot scan load root result")
		}
		root.Pipeline.Type = typePipeline
		if root.Environment.ID == 0 {
			root.Environment = sdk.DefaultEnv
		}

		// Check duplicate pipeline
		var lastTree *sdk.CDPipeline
		if len(cdTrees) > 0 {
			lastTree = &cdTrees[len(cdTrees)-1]
		}

		if lastTree == nil || lastTree.Application.ID != root.Application.ID ||
			lastTree.Pipeline.ID != root.Pipeline.ID || lastTree.Environment.ID != root.Environment.ID {
			if permission.AccessToPipeline(projectkey, root.Environment.Name, root.Pipeline.Name, user, permission.PermissionRead) {
				if hasChild {
					if err := getChild(db, &root, user, branchName, remote, version); err != nil {
						return nil, sdk.WrapError(err, "LoadCDTree> Cannot get child")
					}
				}
				root.Project.Key = projectkey
				root.Application.Permission = permission.ApplicationPermission(projectkey, root.Application.Name, user)
				root.Pipeline.Permission = permission.PipelinePermission(projectkey, root.Pipeline.Name, user)
				if root.Environment.ID != sdk.DefaultEnv.ID {
					root.Environment.Permission = permission.EnvironmentPermission(projectkey, root.Environment.Name, user)
				}

				pipParams, errP := pipeline.GetAllParametersInPipeline(context.TODO(), db, root.Pipeline.ID)
				if errP != nil {
					return nil, sdk.WrapError(err, "LoadCDTree> Cannot get pipeline parameters")
				}
				root.Pipeline.Parameter = pipParams

				// Load Status
				if branchName != "" {
					var pbs []sdk.PipelineBuild
					if version == 0 {
						if root.Pipeline.Type != sdk.BuildPipeline {
							p, errP := project.Load(db, store, projectkey, user, project.LoadOptions.WithEnvironments)
							if errP != nil {
								return nil, sdk.WrapError(errP, "LoadCDTree> Cannot load project")
							}
							for _, e := range p.Environments {

								opts := []pipeline.ExecOptionFunc{
									pipeline.LoadPipelineBuildOpts.WithBranchName(branchName),
								}

								remoteOpts := getRemoteOpts(remote, root.Application.RepositoryFullname)
								if remoteOpts != nil {
									opts = append(opts, remoteOpts)
								}

								builds, errPB := pipeline.LoadPipelineBuildsByApplicationAndPipeline(db, root.Application.ID, root.Pipeline.ID, e.ID, 1, opts...)

								if errPB != nil {
									return nil, sdk.WrapError(errPB, "LoadCDTree> Cannot load last pipeline build for env %s", e.Name)
								}
								pbs = append(pbs, builds...)
							}
						} else {
							opts := []pipeline.ExecOptionFunc{
								pipeline.LoadPipelineBuildOpts.WithBranchName(branchName),
							}

							remoteOpts := getRemoteOpts(remote, root.Application.RepositoryFullname)
							if remoteOpts != nil {
								opts = append(opts, remoteOpts)
							}

							builds, errPB := pipeline.LoadPipelineBuildsByApplicationAndPipeline(db, root.Application.ID, root.Pipeline.ID, root.Environment.ID, 1, opts...)
							if errPB != nil {
								return nil, sdk.WrapError(errPB, "LoadCDTree> Cannot load last pipeline build")
							}
							pbs = builds
						}

					} else {
						if root.Pipeline.Type != sdk.BuildPipeline {
							p, errP := project.Load(db, store, projectkey, user, project.LoadOptions.WithEnvironments)
							if errP != nil {
								return nil, sdk.WrapError(errP, "LoadCDTree> Cannot load project")
							}
							for _, e := range p.Environments {
								builds, errPB := pipeline.LoadPipelineBuildByApplicationPipelineEnvVersion(db, root.Application.ID, root.Pipeline.ID, e.ID, version, 1)
								if errPB != nil {
									return nil, sdk.WrapError(errPB, "LoadCDTree> Cannot load last pipeline build for env %s", e.Name)
								}
								pbs = append(pbs, builds...)
							}
						} else {
							builds, errPB := pipeline.LoadPipelineBuildByApplicationPipelineEnvVersion(db, root.Application.ID, root.Pipeline.ID, root.Environment.ID, version, 1)
							if errPB != nil {
								return nil, sdk.WrapError(errPB, "LoadCDTree> Cannot load last pipeline build")
							}
							pbs = builds
						}
					}

					if len(pbs) > 0 {
						root.Pipeline.LastPipelineBuild = &pbs[0]
						root.Application.PipelinesBuild = pbs
					}
				}

				cdTrees = append(cdTrees, root)
				lastTree = &cdTrees[len(cdTrees)-1]
			}
		}

		if lastTree != nil {
			if err := fetchTriggers(db, lastTree, hasScheduler, hasPoller, hasHook); err != nil {
				return nil, sdk.WrapError(err, "LoadCDTree> Cannot fetch triggers")
			}
		}
	}
	return cdTrees, nil
}

func getChild(db gorp.SqlExecutor, parent *sdk.CDPipeline, user *sdk.User, branchName, remote string, version int64) error {
	listTrigger := []sdk.CDPipeline{}

	query := `
	WITH RECURSIVE parent(id, src_application_id, dest_application_id, src_pipeline_id, src_environment_id, dest_pipeline_id, dest_environment_id) AS (
		SELECT
			pt.id, pt.src_application_id, pt.dest_application_id, pt.src_pipeline_id, COALESCE(pt.src_environment_id,1), pt.dest_pipeline_id, COALESCE(pt.dest_environment_id,1), pt.manual,
			srcPip.name as srcPipName, srcPip.type as srcPipType, destPip.name as destPipName, destPip.type as destpipType,
			srcApp.name as srcAppName, destApp.name as destAppName,
			srcEnv.name as srcEnvName, destEnv.name as destEnvName,
			srcProj.id as srcProjID, srcProj.projectkey as srcProjKey, srcProj.name as srcProjName,
			destProj.id as destProjID, destProj.projectkey as destProjKey, destProj.name as destProjName,
			ptp.name as paramName, ptp.type as paramType, ptp.value as paramValue, ptp.description as paramDescription,
			pre.parameter as prerequisiteName, pre.expected_value as prerequisiteValue
		FROM pipeline_trigger as pt
		JOIN pipeline srcPip ON srcPip.id = src_pipeline_id
		JOIN pipeline destPip ON destPip.id = dest_pipeline_id
		JOIN application srcApp ON srcApp.id = src_application_id
		JOIN application destApp ON destApp.id = dest_application_id
		JOIN environment srcEnv ON srcEnv.id = COALESCE(src_environment_id,1)
		JOIN environment destEnv ON destEnv.id = COALESCE(dest_environment_id,1)
		JOIN project as srcProj ON srcProj.id = srcApp.project_id
		JOIN project as destProj ON destProj.id = destApp.project_id
		LEFT JOIN pipeline_trigger_parameter AS ptp ON ptp.pipeline_trigger_id = pt.id
		LEFT JOIN pipeline_trigger_prerequisite pre ON pre.pipeline_trigger_id = pt.id
		WHERE pt.src_application_id = $1 AND pt.src_pipeline_id = $2 AND COALESCE(pt.src_environment_id,1) = $3
		UNION
			SELECT pt.id, pt.src_application_id, pt.dest_application_id, pt.src_pipeline_id, COALESCE(pt.src_environment_id,1), pt.dest_pipeline_id, COALESCE(pt.dest_environment_id,1), pt.manual,
			srcPip.name as srcPipName, srcPip.type as srcPipType, destPip.name as destPipName, destPip.type as destpipType,
			srcApp.name as srcAppName, destApp.name as destAppName,
			srcEnv.name as srcEnvName, destEnv.name as destEnvName,
			srcProj.id as srcProjID, srcProj.projectkey as srcProjKey, srcProj.name as srcProjName,
			destProj.id as destProjID, destProj.projectkey as destProjKey, destProj.name as destProjName,
			ptp.name as paramName, ptp.type as paramType, ptp.value as paramValue, ptp.description as paramDescription,
			pre.parameter as prerequisiteName, pre.expected_value as prerequisiteValue
			FROM parent pr, pipeline_trigger pt
			JOIN pipeline srcPip ON srcPip.id = src_pipeline_id
			JOIN pipeline destPip ON destPip.id = dest_pipeline_id
			JOIN application srcApp ON srcApp.id = src_application_id
			JOIN application destApp ON destApp.id = dest_application_id
			JOIN environment srcEnv ON srcEnv.id = COALESCE(src_environment_id,1)
			JOIN environment destEnv ON destEnv.id = COALESCE(dest_environment_id,1)
			JOIN project as srcProj ON srcProj.id = srcApp.project_id
			JOIN project as destProj ON destProj.id = destApp.project_id
			LEFT JOIN pipeline_trigger_parameter AS ptp ON ptp.pipeline_trigger_id = pt.id
			LEFT JOIN pipeline_trigger_prerequisite pre ON pre.pipeline_trigger_id = pt.id
			WHERE pt.src_application_id = pr.dest_application_id AND pt.src_pipeline_id = pr.dest_pipeline_id AND COALESCE(pt.src_environment_id,1) = COALESCE(pr.dest_environment_id,1)
	)
	SELECT  parent.id,
		src_application_id, dest_application_id,
		src_pipeline_id, src_environment_id, dest_pipeline_id, dest_environment_id,
		manual,
		srcPipName, srcPipType, destPipName, destPipType,
		srcAppName, destAppName,
		srcEnvName, destEnvName,
		srcProjId, srcProjkey, srcProjName, destProjId, destProjKey, destProjName,
		COALESCE(
			json_agg(json_build_object('name', paramName, 'type', paramType, 'value', paramValue, 'description', paramDescription ))
			FILTER (WHERE paramName IS NOT NULL), '[]'
		),
		COALESCE(
			json_agg(json_build_object('parameter', prerequisiteName, 'expected_value', prerequisiteValue))
			FILTER (WHERE prerequisiteName IS NOT NULL), '[]'
		),
		CASE WHEN COALESCE (count(sc.id), 0)>0 THEN true ELSE false END as hasSchedulers,
		CASE WHEN COALESCE (count(h.id), 0)>0 THEN true ELSE false END as hasHooks,
		CASE WHEN COALESCE (count(p.application_id), 0)>0 THEN true ELSE false END as hasPoller
	FROM parent
	LEFT JOIN pipeline_scheduler sc ON sc.application_id = dest_application_id AND sc.pipeline_id = dest_pipeline_id AND sc.environment_id = dest_environment_id
	LEFT JOIN hook h ON h.application_id = dest_application_id AND h.pipeline_id = dest_pipeline_id
	LEFT JOIN poller p ON p.application_id = dest_application_id AND p.pipeline_id = dest_pipeline_id
	GROUP BY
		parent.id,
		src_application_id, dest_application_id, srcAppName, destAppName,
		src_pipeline_id, dest_pipeline_id,
		src_environment_id, dest_environment_id, srcEnvName, destEnvName,
		srcProjId, destProjId, srcProjkey, destProjKey, srcProjName, destProjName,
		manual, srcpipname, destpipname, srcpiptype, destpiptype
	ORDER BY srcEnvName;
	`
	rows, errQuery := db.Query(query, parent.Application.ID, parent.Pipeline.ID, parent.Environment.ID)
	if errQuery != nil {
		return sdk.WrapError(errQuery, "getChild> Cannot load child for root: %d-%d-%d", parent.Application.ID, parent.Pipeline.ID, parent.Environment.ID)
	}
	defer rows.Close()

	for rows.Next() {
		var child sdk.CDPipeline
		var srcType, destType string
		var params, prerequisites string
		var hasSchedulers, hasHooks, hasPoller bool
		if err := rows.Scan(&child.Trigger.ID,
			&child.Trigger.SrcApplication.ID, &child.Trigger.DestApplication.ID,
			&child.Trigger.SrcPipeline.ID, &child.Trigger.SrcEnvironment.ID, &child.Trigger.DestPipeline.ID, &child.Trigger.DestEnvironment.ID,
			&child.Trigger.Manual,
			&child.Trigger.SrcPipeline.Name, &srcType, &child.Trigger.DestPipeline.Name, &destType,
			&child.Trigger.SrcApplication.Name, &child.Trigger.DestApplication.Name,
			&child.Trigger.SrcEnvironment.Name, &child.Trigger.DestEnvironment.Name,
			&child.Trigger.SrcProject.ID, &child.Trigger.SrcProject.Key, &child.Trigger.SrcProject.Name,
			&child.Trigger.DestProject.ID, &child.Trigger.DestProject.Key, &child.Trigger.DestProject.Name,
			&params, &prerequisites,
			&hasSchedulers, &hasHooks, &hasPoller); err != nil {
			return sdk.WrapError(err, "getChild> Cannot scan child for root: %d-%d-%d", parent.Application.ID, parent.Pipeline.ID, parent.Environment.ID)
		}

		child.Project.Key = child.Trigger.SrcProject.Key
		if permission.AccessToPipeline(child.Project.Key, child.Trigger.DestEnvironment.Name, child.Trigger.DestPipeline.Name, user, permission.PermissionRead) {
			child.Trigger.SrcPipeline.Type = srcType
			child.Trigger.DestPipeline.Type = destType

			child.Project = child.Trigger.DestProject
			child.Application = child.Trigger.DestApplication
			child.Pipeline = child.Trigger.DestPipeline
			child.Environment = child.Trigger.DestEnvironment

			pipParams, errP := pipeline.GetAllParametersInPipeline(context.TODO(), db, child.Pipeline.ID)
			if errP != nil {
				return sdk.WrapError(errP, "getChild> Cannot get pipeline parameters")
			}
			child.Pipeline.Parameter = pipParams

			// Load Status
			var pbs []sdk.PipelineBuild
			var errPB error
			if version == 0 {
				opts := []pipeline.ExecOptionFunc{
					pipeline.LoadPipelineBuildOpts.WithBranchName(branchName),
				}

				if parent.Application.Name == child.Application.Name {
					remoteOpts := getRemoteOpts(remote, parent.Application.RepositoryFullname)
					if remoteOpts != nil {
						opts = append(opts, remoteOpts)
					}
				}

				pbs, errPB = pipeline.LoadPipelineBuildsByApplicationAndPipeline(db, child.Application.ID, child.Pipeline.ID, child.Environment.ID, 1, opts...)
			} else {
				pbs, errPB = pipeline.LoadPipelineBuildByApplicationPipelineEnvVersion(db, child.Application.ID, child.Pipeline.ID, child.Environment.ID, version, 1)
			}
			if errPB != nil {
				return sdk.WrapError(errPB, "LoadCDTree> Cannot load last pipeline build")
			}
			if len(pbs) > 0 {
				child.Pipeline.LastPipelineBuild = &pbs[0]
			}

			// Calculate permission
			child.Application.Permission = permission.ApplicationPermission(child.Project.Key, child.Application.Name, user)
			child.Project.Permission = permission.ProjectPermission(child.Project.Key, user)
			child.Pipeline.Permission = permission.PipelinePermission(child.Project.Key, child.Pipeline.Name, user)
			child.Environment.Permission = permission.EnvironmentPermission(child.Project.Key, child.Pipeline.Name, user)

			if err := json.Unmarshal([]byte(params), &child.Trigger.Parameters); err != nil {
				return sdk.WrapError(err, "getChild> Cannot unmarshal trigger params")
			}

			if err := json.Unmarshal([]byte(prerequisites), &child.Trigger.Prerequisites); err != nil {
				return sdk.WrapError(err, "getChild> Cannot unmarshal trigger prerequisite")
			}

			if err := fetchTriggers(db, &child, hasSchedulers, hasPoller, hasHooks); err != nil {
				return sdk.WrapError(err, "getChild> Cannot fetch trigger")
			}

			listTrigger = append(listTrigger, child)
		}
	}

	buildTreeOrder(parent, listTrigger, user)
	return nil
}

func getRemoteOpts(remoteFilter, repoFullName string) pipeline.ExecOptionFunc {
	if repoFullName != "" && (remoteFilter == "" || remoteFilter == repoFullName) {
		return pipeline.LoadPipelineBuildOpts.WithEmptyRemote(repoFullName)
	} else if repoFullName != "" && remoteFilter != "" {
		return pipeline.LoadPipelineBuildOpts.WithRemoteName(remoteFilter)
	}

	return nil
}

func buildTreeOrder(parent *sdk.CDPipeline, listChild []sdk.CDPipeline, user *sdk.User) sdk.CDPipeline {

	for _, child := range listChild {
		if child.Trigger.SrcProject.ID == parent.Project.ID &&
			child.Trigger.SrcApplication.ID == parent.Application.ID &&
			child.Trigger.SrcPipeline.ID == parent.Pipeline.ID &&
			child.Trigger.SrcEnvironment.ID == parent.Environment.ID {

			child.Application.Permission = permission.ApplicationPermission(child.Project.Key, child.Application.Name, user)
			child.Pipeline.Permission = permission.PipelinePermission(child.Project.Key, child.Pipeline.Name, user)
			if child.Environment.ID != sdk.DefaultEnv.ID {
				child.Environment.Permission = permission.EnvironmentPermission(child.Project.Key, child.Environment.Name, user)
			}
			parent.SubPipelines = append(parent.SubPipelines, buildTreeOrder(&child, listChild, user))
		}
	}
	return *parent
}

func fetchTriggers(db gorp.SqlExecutor, workflowItem *sdk.CDPipeline, hasSchedulers, hasPoller, hasHooks bool) error {
	if hasHooks {
		hooks, errH := hook.LoadPipelineHooks(db, workflowItem.Pipeline.ID, workflowItem.Application.ID)
		if errH != nil {
			return sdk.WrapError(errH, "fetchTriggers> Cannot load hooks for application %s [%d] and pipeline %s [%d]: %s", workflowItem.Application.Name, workflowItem.Application.ID, workflowItem.Pipeline.Name, workflowItem.Pipeline.ID, errH)
		}
		workflowItem.Hooks = hooks
	}
	if hasPoller {
		poller, errP := poller.LoadByApplicationAndPipeline(db, workflowItem.Application.ID, workflowItem.Pipeline.ID)
		if errP != nil {
			return sdk.WrapError(errP, "fetchTriggers> Cannot load pollers for application %s [%d] and pipeline %s [%d]: %s", workflowItem.Application.Name, workflowItem.Application.ID, workflowItem.Pipeline.Name, workflowItem.Pipeline.ID, errP)
		}
		workflowItem.Poller = poller
	}

	if hasSchedulers {
		schedulers, errS := scheduler.GetByApplicationPipeline(db, &workflowItem.Application, &workflowItem.Pipeline)
		if errS != nil {
			return sdk.WrapError(errS, "fetchTriggers> Cannot load schedulers for application %s [%d] and pipeline %s [%d]: %s", workflowItem.Application.Name, workflowItem.Application.ID, workflowItem.Pipeline.Name, workflowItem.Pipeline.ID, errS)
		}
		workflowItem.Schedulers = schedulers
	}
	return nil
}
