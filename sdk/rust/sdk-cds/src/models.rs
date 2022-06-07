//! All models about CDS
use chrono::prelude::*;
use serde_json;
use std::collections::HashMap;

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct AuthConsumerSigninResponse {
    pub api_url: String,
    pub token: String,
    pub user: Option<AuthentifiedUser>,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct AuthentifiedUser {
    pub id: String,
    pub created: Option<DateTime<Utc>>,
    pub username: String,
    pub fullname: String,
    pub ring: String,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct StatusLine {
    pub status: String,
    pub component: String,
    pub value: String,
    #[serde(rename = "type")]
    pub _type: String,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct MonitoringStatus {
    pub now: Option<DateTime<Utc>>,
    pub lines: Option<Vec<StatusLine>>,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Group {
    pub id: i64,
    pub name: String,
    pub admins: Option<Vec<User>>,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Permissions {
    #[serde(rename = "Groups")]
    pub groups: Option<Vec<String>>,
    #[serde(rename = "ProjectsPerm")]
    pub projects_perm: Option<HashMap<String, u8>>,
    #[serde(rename = "WorkflowsPerm")]
    pub workflows_perm: Option<HashMap<String, u8>>,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct User {
    pub id: String,
    pub username: String,
    pub fullname: String,
    pub ring: String,
    pub created: Option<DateTime<Utc>>,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Project {
    pub key: String,
    pub name: String,
    pub description: String,
    pub icon: String,
    pub permissions: HashMap<String, bool>,
    pub created: String,
    pub last_modified: String,
    pub metadata: Option<serde_json::Value>,
    pub keys: Option<Vec<Key>>,
    pub vcs_servers: Option<Vec<VcsServer>>,
    pub integrations: Option<Vec<Integration>>,
    pub features: serde_json::Value,
    pub favorite: bool,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Integration {
    pub id: i64,
    pub project_id: i64,
    pub name: String,
    pub integration_model_id: i64,
    pub model: IntegrationModel,
    pub config: Option<HashMap<String, ConfigValue>>,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct IntegrationModel {
    pub id: i64,
    pub name: String,
    pub author: String,
    pub identifier: String,
    pub icon: String,
    pub default_config: Option<HashMap<String, ConfigValue>>,
    pub deployment_default_config: Option<HashMap<String, ConfigValue>>,
    pub disabled: bool,
    pub hook: bool,
    pub file_storage: bool,
    pub block_storage: bool,
    pub deployment: bool,
    pub compute: bool,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct ConfigValue {
    pub value: String,
    #[serde(rename = "type")]
    pub _type: String,
    pub description: String,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct VcsServer {
    pub name: String,
    pub username: String,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Application {
    pub id: i64,
    pub name: String,
    pub description: String,
    pub icon: String,
    pub project_key: String,
    pub permission: i64,
    pub last_modified: String,
    pub vcs_server: String,
    pub repository_fullname: String,
    pub vcs_strategy: VcsStrategy,
    pub metadata: serde_json::Value,
    pub keys: Option<Vec<Key>>,
    pub deployment_strategies: serde_json::Value,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct VcsStrategy {
    pub connection_type: String,
    pub ssh_key: String,
    pub user: String,
    pub password: String,
    pub pgp_key: String,
    pub branch: String,
    pub default_branch: String,
    pub ssh_key_content: String,
}

// workflow

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Audit {
    pub id: i64,
    pub triggered_by: String,
    pub created: String,
    pub data_before: String,
    pub data_after: String,
    pub event_type: String,
    pub data_type: String,
    pub project_key: String,
    pub workflow_id: i64,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Conditions {
    pub plain: Option<Vec<PlainCondition>>,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Context {
    pub id: i64,
    pub node_id: i64,
    pub pipeline_id: i64,
    pub application_id: i64,
    pub environment_id: i64,
    pub project_integration_id: Option<i64>,
    pub default_payload: serde_json::Value,
    // default_pipeline_parameters: string,
    pub conditions: Conditions,
    pub mutex: bool,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Environment {
    pub id: i64,
    pub name: String,
    pub variables: Option<Vec<Variable>>,
    pub project_key: String,
    pub permission: i64,
    pub last_modified: i64,
    pub keys: Option<Vec<Key>>,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Key {
    pub name: String,
    pub public: String,
    pub private: String,
    pub key_id: String,
    #[serde(rename = "type")]
    pub _type: String,
    pub application_id: i64,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Node {
    pub id: i64,
    pub workflow_id: i64,
    pub name: String,
    #[serde(rename = "ref")]
    pub _ref: String,
    #[serde(rename = "type")]
    pub _type: String,
    pub triggers: Option<Vec<Trigger>>,
    pub context: Option<Context>,
    pub outgoing_hook: Option<Hook>,
    pub parents: Option<Vec<Parent>>,
    pub hooks: Option<Vec<Hook>>,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Parent {
    pub id: i64,
    pub node_id: i64,
    pub parent_name: String,
    pub parent_id: i64,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Pipeline {
    pub id: i64,
    pub name: String,
    pub description: String,
    #[serde(rename = "type")]
    pub _type: String,
    #[serde(rename = "projectKey")]
    pub project_key: String,
    //   stages: string,
    pub permission: i64,
    pub last_modified: i64,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct PlainCondition {
    pub variable: String,
    pub operator: String,
    pub value: String,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Workflow {
    pub id: i64,
    pub name: String,
    pub last_modified: String,
    pub project_id: i64,
    pub project_key: String,
    pub root_id: i64,
    pub permission: i64,
    pub metadata: serde_json::Value,
    pub usage: Usage,
    pub history_length: i64,
    pub audits: Option<Vec<Audit>>,
    pub pipelines: Option<HashMap<i64, Pipeline>>,
    pub applications: Option<HashMap<i64, Application>>,
    pub environments: Option<HashMap<i64, Environment>>,
    // project_platforms: (),
    //   labels: string,
    pub to_delete: bool,
    pub favorite: bool,
    pub workflow_data: WorkflowData,
    //   as_code_events: Vec<()>,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Hook {
    pub id: i64,
    pub uuid: String,
    #[serde(rename = "ref")]
    pub _ref: String,
    pub node_id: i64,
    pub hook_model_id: i64,
    pub config: serde_json::Value,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct HookModel {
    pub id: i64,
    pub name: String,
    #[serde(rename = "type")]
    pub _type: String,
    pub author: String,
    pub description: String,
    pub identifier: String,
    pub icon: String,
    pub command: String,
    pub default_config: serde_json::Value,
    pub disabled: bool,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Trigger {
    pub id: i64,
    pub parent_node_id: i64,
    pub child_node_id: i64,
    pub parent_node_name: String,
    pub child_node: Node,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Usage {
    pub environments: Option<Vec<Environment>>,
    pub pipelines: Option<Vec<Pipeline>>,
    pub applications: Option<Vec<Application>>,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Variable {
    pub id: i64,
    pub name: String,
    pub value: String,
    #[serde(rename = "type")]
    pub _type: String,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct WorkflowData {
    pub node: Node,
    pub joins: Option<Vec<Node>>,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct QueueCount {
    #[serde(rename = "version")]
    pub count: i64,
    pub since: String,
    pub until: String,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Bookmark {
    pub icon: String,
    pub description: String,
    pub key: String,
    pub name: String,
    pub application_name: String,
    pub workflow_name: String,
    #[serde(rename = "type")]
    pub _type: String,
    pub favorite: bool,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Action {
    pub id: i64,
    pub name: String,
    pub step_name: String,
    #[serde(rename = "type")]
    pub _type: String,
    pub description: String,
    // requirements: string,
    pub parameters: Option<Vec<Parameter>>,
    pub action: Option<Vec<Action>>,
    pub enabled: bool,
    pub deprecated: bool,
    pub optional: bool,
    pub always_executed: bool,
    pub last_modified: i64,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Bookedby {
    pub id: i64,
    pub name: String,
    #[serde(rename = "type")]
    pub _type: String,
    pub http_url: String,
    pub last_heartbeat: String,
    pub hash: String,
    pub token: String,
    pub group_id: Option<i64>,
    pub is_shared_infra: bool,
    pub version: String,
    pub up_to_date: bool,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct WorkflowRun {
    pub id: i64,
    pub num: i64,
    pub project_id: i64,
    pub workflow_id: i64,
    pub status: String,
    pub workflow: Workflow,
    pub start: String,
    pub last_modified: String,
    pub nodes: HashMap<String, Vec<NodeRun>>,
    // infos: Vec<Infos>,
    pub tags: Vec<Tag>,
    pub last_subnumber: i64,
    pub last_execution: String,
    pub to_delete: bool,
    pub header: Option<HashMap<String, String>>,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Tag {
    pub tag: String,
    pub value: String,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct NodeRun {
    pub workflow_run_id: i64,
    pub workflow_id: i64,
    pub application_id: i64,
    pub id: i64,
    pub workflow_node_id: i64,
    pub workflow_node_name: String,
    pub num: i64,
    pub subnumber: i64,
    pub status: String,
    pub stages: Option<Vec<Stages>>,
    pub start: String,
    pub last_modified: String,
    pub done: String,
    pub manual: ManualRequest,
    pub payload: serde_json::Value,
    pub build_parameters: Option<Vec<Parameter>>,
    pub coverage: Coverage,
    // pub vulnerabilities_report: VulnerabilitiesReport,
    pub vcs_repository: String,
    pub vcs_tag: String,
    pub vcs_branch: String,
    pub vcs_hash: String,
    pub vcs_server: String,
    pub can_be_run: bool,
    pub header: Option<HashMap<String, String>>,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct ManualRequest {
    pub payload: serde_json::Value,
    pub pipeline_parameter: Option<Vec<Parameter>>,
    pub user: User,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Stages {
    pub id: i64,
    pub name: String,
    pub build_order: i64,
    pub enabled: bool,
    pub run_jobs: Option<Vec<RunJob>>,
    //   prerequisites: string,
    pub last_modified: i64,
    pub jobs: Vec<Job>,
    pub status: String,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct RunJob {
    pub project_id: i64,
    pub id: i64,
    pub workflow_node_run_id: i64,
    pub job: Job,
    pub parameters: Option<Vec<Parameter>>,
    pub status: String,
    pub retry: i64,
    pub queued: String,
    pub queued_seconds: i64,
    pub start: String,
    pub done: String,
    pub bookedby: Bookedby,
    //   spawninfos: Vec<Spawninfos>,
    pub exec_groups: Vec<Group>,
    pub header: Option<HashMap<String, String>>,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Coverage {
    pub workflow_id: i64,
    pub workflow_node_run_id: i64,
    pub workflow_run_id: i64,
    pub application_id: i64,
    pub run_number: i64,
    pub repository: String,
    pub branch: String,
    pub report: Report,
    // pub trend: Trend,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Job {
    // pub step_status: string,
    pub reason: String,
    pub worker_name: String,
    pub worker_id: String,
    pub pipeline_action_id: i64,
    pub pipeline_stage_id: i64,
    pub enabled: bool,
    pub last_modified: i64,
    pub action: Action,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Parameter {
    pub id: i64,
    pub name: String,
    #[serde(rename = "type")]
    pub _type: String,
    pub value: String,
    pub description: String,
}

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Report {
    //   files: string,
    pub total_lines: i64,
    pub covered_lines: i64,
    pub total_functions: i64,
    pub covered_functions: i64,
    pub total_branches: i64,
    pub covered_branches: i64,
}
