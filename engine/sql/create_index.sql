-- ACTION REQUIREMENT
select create_index('action_requirement', 'IDX_ACTION_REQUIREMENT_ACTION_ID', 'action_id');

-- ACTION_EDGE
select create_index('action_edge', 'IDX_ACTION_EDGE_PARENT_ID', 'parent_id');
select create_index('action_edge', 'IDX_ACTION_EDGE_CHILD_ID', 'child_id');

-- ACTION EDGE PARAMETER
select create_index('action_edge_parameter', 'IDX_ACTION_EDGE_PARAMETER_ACTION_EDGE_ID', 'action_edge_id');

-- ACTION PARAMETER
select create_unique_index('action_parameter', 'IDX_ACTION_PARAMETER_ACTION_ID', 'action_id,name');

-- PIPELINE BUILD
select create_index('pipeline_build', 'IDX_PIPELINE_BUILD_UNIQUE_BUILD_NUMBER', 'build_number,pipeline_id,application_id,environment_id');

-- ACTION BUILD
select create_index('action_build', 'IDX_ACTION_BUILD_PIPELINE_BUILD_ID', 'pipeline_build_id');
select create_index('action_build', 'IDX_ACTION_BUILD_PIPELINE_ACTION_ID', 'pipeline_action_id');
select create_unique_index('action_build', 'IDX_ACTION_BUILD_PIPELINE_ACTION_ID_BUILD_ID', 'pipeline_build_id,pipeline_action_id');

-- ARTIFACT
select create_index('artifact', 'IDX_ARTIFACT_PIPELINE_ID', 'pipeline_id');
select create_index('artifact', 'IDX_ARTIFACT_APPLICATION_ID', 'application_id');
select create_index('artifact','IDX_ARTIFACT_ENVIRONMENT', 'environment_id');

-- APPLICATION
select create_unique_index('application', 'IDX_APPLICATION_PROJECT_ID_NAME', 'project_id,name');

-- APPLICATION PIPELINE
select create_unique_index('application_pipeline','IDX_APPLICATION_PIPELINE_APPLICATION', 'application_id,pipeline_id');

-- BUILD_LOG
select create_index('build_log', 'IDX_ARTIFACT_ACTION_BUILD_ID', 'action_build_id');

-- ENVIRONMENT
select create_unique_index('environment','IDX_ENVIRONMENT', 'name,project_id');

-- ENVIRONMENT_GROUP
select create_unique_index('environment_group','IDX_ENVIRONMENT_GROUP_UNIQUE', 'group_id,environment_id');
select create_index('environment_group','IDX_ENVIRONMENT_GROUP_GROUP', 'group_id');
select create_index('environment_group','IDX_ENVIRONMENT_GROUP_ENV', 'environment_id');

-- ENVIRONMENT_VARIABLE
select create_unique_index('environment_variable','IDX_ENVIRONMENT_VARIABLE_ID_NAME', 'environment_id,name');

-- GROUP
select create_unique_index('group', 'IDX_GROUP_NAME', 'name');

-- HOOK
select create_index('hook','IDX_HOOK_PIPELINE_ID','pipeline_id');

-- PIPELINE
select create_unique_index('pipeline','IDX_PIPELINE_NAME','name,project_id');
select create_index('pipeline','IDX_PIPELINE_PROJECT_ID','project_id');

-- PIPELINE_ACTION
select create_index('pipeline_action','IDX_PIPELINE_ACTION_STAGE_ID','pipeline_stage_id');
select create_index('pipeline_action','IDX_PIPELINE_ACTION_ACTION_ID','action_id');

-- PIPELINE BUILD
select create_index('pipeline_build','IDX_PIPELINE_BUILD_PIPELINE_ID','pipeline_id');
select create_index('pipeline_build','IDX_PIPELINE_BUILD_NUMBER','build_number');
select create_index('pipeline_build','IDX_PIPELINE_BUILD_APPLICATION_ID','application_id');
select create_index('pipeline_build','IDX_PIPELINE_BUILD_ENVIRONMENT_ID','environment_id');

-- PIPELINE PARAMETER
select create_unique_index('pipeline_parameter','IDX_PIPELINE_PARAMETER_NAME','pipeline_id,name');

-- PIPELINE HISTORY
select create_index('pipeline_history', 'IDX_PIPELINE_HISTORY', 'build_number');
select create_index('pipeline_history','IDX_PIPELINE_HISTORY_ENVIRONMENT', 'environment_id');

-- PIPELINE Stage
select create_index('pipeline_stage','IDX_PIPELINE_STAGE_BUILD_ORDER','build_order');
select create_index('pipeline_stage','IDX_PIPELINE_STAGE_PIPELINE_ID','pipeline_id');

-- PIPELINE TRIGGER
select create_index('pipeline_trigger','IDX_PIPELINE_TRIGGER_SRC_APPLICATION', 'src_application_id');
select create_index('pipeline_trigger','IDX_PIPELINE_TRIGGER_DEST_APPLICATION', 'dest_application_id');
select create_index('pipeline_trigger','IDX_PIPELINE_TRIGGER_SRC_PIPELINE', 'src_pipeline_id');
select create_index('pipeline_trigger','IDX_PIPELINE_TRIGGER_DEST_PIPELINE', 'dest_pipeline_id');
select create_index('pipeline_trigger','IDX_PIPELINE_TRIGGER_SRC_ENVIRONMENT', 'src_environment_id');
select create_index('pipeline_trigger','IDX_PIPELINE_TRIGGER_DEST_ENVIRONMENT', 'dest_environment_id');

-- PLUGIN
select create_unique_index('plugin','IDX_PLUGIN_NAME', 'name');

-- PROJECT
select create_unique_index('project','IDX_PROJECT_KEY','projectKey');

-- SYSTEM_LOG
select create_index('system_log','IDX_SYS_LOG_LOGGED','logged');

-- USER
select create_unique_index('user','IDX_USER_USERNAME','username');

-- USER KEY
select create_index('user_key','IDX_USER_KEY_USER_KEY','user_key');

-- WORKER
select create_index('worker','IDX_WORKER_ID','id');
select create_index('worker','IDX_WORKER_OWNER_ID','owner_id');

-- WORKER_CAPABILITY
select create_unique_index('worker_capability','IDX_WORKER_CAPABILITY_NAME','worker_model_id,name');
select create_index('worker_capability','IDX_WORKER_CAPABILITY_MODEL_ID','worker_model_id');

-- WORKER_MODEL
select create_unique_index('worker_model','IDX_WORKER_MODEL_NAME','name');
select create_index('worker_model','IDX_WORKER_MODEL_OWNER_ID','owner_id');

-- REPOSITORIES_MANAGER_PROJECT
select create_unique_index('repositories_manager_project', 'IDX_REPOSITORIES_MANAGER_PROJECT_ID' ,'id_repositories_manager, id_project');
