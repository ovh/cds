-- ACTION BUILD
select create_foreign_key('FK_ACTION_BUILD_PIPELINE_ACTION', 'action_build', 'pipeline_action', 'pipeline_action_id', 'id');
select create_foreign_key('FK_ACTION_BUILD_PIPELINE_BUILD', 'action_build', 'pipeline_build', 'pipeline_build_id', 'id');

-- ACTION REQUIREMENT
select create_foreign_key('FK_ACTION_REQUIREMENT_ACTION', 'action_requirement', 'action', 'action_id', 'id');

-- ACTION EDGE
select create_foreign_key('FK_ACTION_EDGE_PARENT_ACTION', 'action_edge', 'action', 'parent_id', 'id');
select create_foreign_key('FK_ACTION_EDGE_CHILD_ACTION', 'action_edge', 'action', 'child_id', 'id');

-- ACTION EDGE PARAMETER
select create_foreign_key('FK_ACTION_EDGE_PARAMETER_ACTION_EDGE', 'action_edge_parameter', 'action_edge', 'action_edge_id', 'id');

-- ACTION PARAMETER
select create_foreign_key('FK_ACTION_PARAMETER_ACTION', 'action_parameter', 'action', 'action_id', 'id');

-- ARTIFACT
select create_foreign_key('FK_ARTIFACT_PIPELINE_BUILD', 'artifact', 'pipeline', 'pipeline_id', 'id');
select create_foreign_key('FK_ARTIFACT_APPLICATION', 'artifact', 'application', 'application_id', 'id');
select create_foreign_key('FK_ARTIFACT_ENVIRONMENT', 'artifact', 'environment', 'environment_id', 'id');

-- APPLICATION
select create_foreign_key('FK_APPLICATION_PROJECT', 'application', 'project', 'project_id', 'id');
select create_foreign_key('FK_APPLICATION_REPOSITORIES_MANAGER', 'application', 'repositories_manager', 'repository_manager_id', 'id');

-- APPLICATION GROUP
select create_foreign_key('FK_APPLICATION_GROUP_APPLICATION', 'application_group', 'application', 'application_id', 'id');
select create_foreign_key('FK_APPLICATION_GROUP_GROUP', 'application_group', 'group', 'group_id', 'id');

-- APPLICATION PIPELINE
select create_foreign_key('FK_APPLICATION_PIPELINE_APPLICATION', 'application_pipeline', 'application', 'application_id', 'id');
select create_foreign_key('FK_APPLICATION_PIPELINE_PIPELINE', 'application_pipeline', 'pipeline', 'pipeline_id', 'id');

-- APPLICATION_VARIABLE
select create_foreign_key('FK_APPLICATION_VARIABLE_APPLICATION', 'application_variable', 'application', 'application_id', 'id');

-- APPLICATION PIPELINE NOTIF
SELECT create_foreign_key('FK_APPLICATION_PIPELINE_NOTIF_APPLICATION_PIPELINE', 'application_pipeline_notif', 'application_pipeline', 'application_pipeline_id', 'id');
SELECT create_foreign_key('FK_APPLICATION_PIPELINE_NOTIF_ENVIRONMENT', 'application_pipeline_notif', 'environment', 'environment_id', 'id');

-- BUILD_LOG
select create_foreign_key('FK_BUILD_LOG_ACTION_BUILD', 'build_log', 'action_build', 'action_build_id', 'id');

-- ENVIRONMENT
select create_foreign_key('FK_ENVIRONMENT_PROJECT', 'environment', 'project', 'project_id', 'id');

-- ENVIRONMENT_GROUP
select create_foreign_key('FK_ENVIRONMENT_GROUP_ENV', 'environment_group', 'environment', 'environment_id', 'id');
select create_foreign_key('FK_ENVIRONMENT_GROUP_GROUP', 'environment_group', 'group', 'group_id', 'id');

-- ENVIRONMENT_VARIABLE
select create_foreign_key('FK_ENVIRONMENT_VARIABLE_ENV', 'environment_variable', 'environment', 'environment_id', 'id');

-- GROUP USER
select create_foreign_key('FK_GROUP_USER_GROUP', 'group_user', 'group', 'group_id', 'id');
select create_foreign_key('FK_GROUP_USER_USER', 'group_user', 'user', 'user_id', 'id');

-- HOOK
select create_foreign_key('FK_HOOK_PIPELINE', 'hook', 'pipeline', 'pipeline_id', 'id');
select create_foreign_key('FK_HOOK_APPLICATION', 'hook', 'application', 'application_id', 'id');

-- PIPELINE
select create_foreign_key('FK_PIPELINE_PROJECT', 'pipeline', 'project', 'project_id', 'id');

-- PIPELINE ACTION
select create_foreign_key('FK_PIPELINE_ACTION_ACTION', 'pipeline_action', 'action', 'action_id', 'id');
select create_foreign_key('FK_PIPELINE_ACTION_PIPELINE_STAGE', 'pipeline_action', 'pipeline_stage', 'pipeline_stage_id', 'id');

-- PIPELINE BUILD
select create_foreign_key('FK_PIPELINE_BUILD_PIPELINE', 'pipeline_build', 'pipeline', 'pipeline_id', 'id');
select create_foreign_key('FK_PIPELINE_BUILD_APPLICATION', 'pipeline_build', 'application', 'application_id', 'id');
select create_foreign_key('FK_PIPELINE_BUILD_ENVIRONMENT', 'pipeline_build', 'environment', 'environment_id', 'id');

-- PIPELINE GROUP
select create_foreign_key('FK_PIPELINE_GROUP_PIPELINE', 'pipeline_group', 'pipeline', 'pipeline_id', 'id');
select create_foreign_key('FK_PIPELINE_GROUP', 'pipeline_group', 'group', 'group_id', 'id');

-- PIPELINE HISTORY
select create_foreign_key('FK_PIPELINE_HISTORY_PIPELINE', 'pipeline_history', 'pipeline', 'pipeline_id', 'id');
select create_foreign_key('FK_PIPELINE_HISTORY_ENVIRONMENT', 'pipeline_history', 'environment', 'environment_id', 'id');

-- PIPELINE STAGE
select create_foreign_key('FK_PIPELINE_STAGE_PIPELINE', 'pipeline_stage', 'pipeline', 'pipeline_id', 'id');

-- PIPELINE STAGE PREREQUISITE
select create_foreign_key('FK_PIPELINE_STAGE_PREREQUISITE_PIPELINE_STAGE', 'pipeline_stage_prerequisite', 'pipeline_stage', 'pipeline_stage_id', 'id');

-- PIPELINE PARAMETER
select create_foreign_key('FK_PIPELINE_PARAMETER_PIPELINE', 'pipeline_parameter', 'pipeline', 'pipeline_id', 'id');

-- PIPELINE TRIGGER
select create_foreign_key('FK_PIPELINE_TRIGGER_SRC_APPLICATION', 'pipeline_trigger', 'application', 'src_application_id', 'id');
select create_foreign_key('FK_PIPELINE_TRIGGER_DEST_APPLICATION', 'pipeline_trigger', 'application', 'dest_application_id', 'id');
select create_foreign_key('FK_PIPELINE_TRIGGER_SRC_PIPELINE', 'pipeline_trigger', 'pipeline', 'src_pipeline_id', 'id');
select create_foreign_key('FK_PIPELINE_TRIGGER_DEST_PIPELINE', 'pipeline_trigger', 'pipeline', 'dest_pipeline_id', 'id');
select create_foreign_key('FK_PIPELINE_TRIGGER_SRC_ENVIRONMENT', 'pipeline_trigger', 'environment', 'src_environment_id', 'id');
select create_foreign_key('FK_PIPELINE_TRIGGER_DEST_ENVIRONMENT', 'pipeline_trigger', 'environment', 'dest_environment_id', 'id');

-- PIPELINE TRIGGER PARAMETER
select create_foreign_key('FK_PIPELINE_TRIGGER_PARAMETER_PIPELINE', 'pipeline_trigger_parameter', 'pipeline_trigger', 'pipeline_trigger_id', 'id');

-- PIPELINE TRIGGER PREREQUISITE
select create_foreign_key('FK_PIPELINE_TRIGGER_PREREQUISITE_PIPELINE_TRIGGER', 'pipeline_trigger_prerequisite', 'pipeline_trigger', 'pipeline_trigger_id', 'id');

-- POLLER
select create_foreign_key('FK_POLLER_APPLICATION', 'poller', 'application', 'application_id', 'id');
select create_foreign_key('FK_POLLER_PIPELINE', 'poller', 'pipeline', 'pipeline_id', 'id');

-- POLLER EXECUTION
select create_foreign_key('FK_POLLER_EXECUTION_APPLICATION', 'poller_execution', 'application', 'application_id', 'id');
select create_foreign_key('FK_POLLER_EXECUTION_PIPELINE', 'poller_execution', 'pipeline', 'pipeline_id', 'id');

-- PROJECT GROUP
select create_foreign_key('FK_PROJECT_GROUP_PIPELINE', 'project_group', 'project', 'project_id', 'id');
select create_foreign_key('FK_PROJECT_GROUP', 'project_group', 'group', 'group_id', 'id');

-- PROJECT VARIABLE
select create_foreign_key('FK_PROJECT_VARIABLE_PIPELINE', 'project_variable', 'project', 'project_id', 'id');

-- USER KEY
select create_foreign_key('FK_USER_KEY_USER', 'user_key', 'user', 'user_id', 'id');

-- WORKER CAPABILITY
select create_foreign_key('FK_WORKER_CAPABILITY_WORKER_MODEL', 'worker_capability', 'worker_model', 'worker_model_id', 'id');

-- WORKER
select create_foreign_key('FK_WORKER_ACTION_BUILD', 'worker', 'action_build', 'action_build_id', 'id');

-- HATCHERY
select create_foreign_key('FK_HATCHERY_OWNER_ID', 'hatchery', 'user', 'owner_id', 'id');

-- HATCHERY MODEL
select create_foreign_key('FK_HATCHERY_MODEL_HATCHERY_ID', 'hatchery_model', 'hatchery', 'hatchery_id', 'id');
select create_foreign_key('FK_HATCHERY_MODEL_WORKER_MODEL_ID', 'hatchery_model', 'worker_model', 'worker_model_id', 'id');

-- repositories_manager_project
select create_foreign_key('fk_repositories_manager_project_repositories_manager_id', 'repositories_manager_project', 'repositories_manager', 'id_repositories_manager', 'id')
select create_foreign_key('fk_repositories_manager_project_project_id', 'repositories_manager_project', 'project', 'id_project', 'id')

-- warning
ALTER TABLE warning ADD CONSTRAINT fk_application FOREIGN KEY (app_id) references application (id) ON delete cascade;
ALTER TABLE warning ADD CONSTRAINT fk_pipeline FOREIGN KEY (pip_id) references pipeline (id) ON delete cascade;
ALTER TABLE warning ADD CONSTRAINT fk_environment FOREIGN KEY (env_id) references environment (id) ON delete cascade;
ALTER TABLE warning ADD CONSTRAINT fk_action FOREIGN KEY (action_id) references action (id) ON delete cascade;

-- AUDIT
ALTER TABLE project_variable_audit ADD CONSTRAINT fk_project FOREIGN KEY (project_id) references project (id) ON delete cascade;
ALTER TABLE application_variable_audit ADD CONSTRAINT fk_application FOREIGN KEY (application_id) references application (id) ON delete cascade;
ALTER TABLE environment_variable_audit ADD CONSTRAINT fk_environment FOREIGN KEY (environment_id) references environment (id) ON delete cascade;
