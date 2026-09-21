import { Permission } from 'app/model/permission.model';
import { Application } from './application.model';
import { Environment } from './environment.model';
import { GroupPermission } from './group.model';
import { ProjectIntegration } from './integration.model';
import { Key } from './keys.model';
import { Pipeline } from './pipeline.model';
import { Variable } from './variable.model';
import { Workflow } from './workflow.model';
import { VariableSet } from './variablesets.model';
import { VCSProject } from './vcs.model';
import { Initiator } from './analysis.model';

export class Project {
  key: string;
  name: string;
  description: string;
  icon: string;
  workflows: Array<Workflow>;
  workflow_names: Array<IdName>;
  pipelines: Array<Pipeline>;
  pipeline_names: Array<IdName>;
  applications: Array<Application>;
  application_names: Array<IdName>;
  groups: Array<GroupPermission>;
  variables: Array<Variable>;
  variablesets: Array<VariableSet>;
  environments: Array<Environment>;
  environment_names: Array<IdName>;
  permissions: Permission;
  last_modified: string;
  vcs_servers: Array<VCSProject>;
  keys: Array<Key>;
  integrations: Array<ProjectIntegration>;
  features: {};
  labels: Label[];
  metadata: {};
  workflow_retention: number;
  // true if someone has updated the project ( used for warnings )
  externalChange: boolean;
  loading: boolean;
  mute: boolean;
  organization: string;
}

export class LoadOpts {
  constructor(
    public queryParam: string,
    public fieldName: string
  ) { }
}

export class IdName {
  id: number;
  name: string;
  description?: string;
  icon?: string;
  labels?: Label[];
  // ui params
  mute: boolean;
}

export class Label {
  id: number;
  name: string;
  color: string;
  project_id: number;
  workflow_id: number;
  // ui params
  font_color: string;
}

export class ProjectRepository {
  id: string;
  name: string;
  clone_url: string;
  created: Date;
  created_by: string;
  // Listened by a workflow of the project without being declared in it: only its name is known
  distant?: boolean;
}

export class ProjectDistantRepository {
  vcs_name: string;
  repository: string;
  // The workflows of the project that listen to it
  workflows: Array<ProjectDistantRepositoryWorkflow>;
}

export class ProjectDistantRepositoryWorkflow {
  vcs_name: string;
  repository_name: string;
  workflow_name: string;
}

export enum WorkflowHookEventName {
  WorkflowHookEventNameWorkflowUpdate = "workflow-update",
  WorkflowHookEventNameModelUpdate = "model-update",
  WorkflowHookEventNamePush = "push",
  WorkflowHookEventNameManual = "manual",
  WorkflowHookEventNameScheduler = "scheduler",
  WorkflowHookEventNamePullRequest = "pull-request",
  WorkflowHookEventNamePullRequestComment = "pull-request-comment",
  WorkflowHookEventNamePullWorkflowRun = "workflow-run"

}

export class RepositoryHookEvent {
  uuid: string;
  created: number;
  status: string;
  event_name: WorkflowHookEventName;
  event_type: string;
  extracted_data: RepositoryHookEventExtractedData;
  /** Who the event is attributed to; `username` is what older events carry instead. */
  initiator: Initiator;
  username: string;
  last_error: string;
  vcs_server_name: string;
  repository_name: string;
  analyses: Array<RepositoryHookEventAnalysis>;
  workflows: Array<RepositoryHookWorkflow>;
  sign_key: string;

  // UI data
  nbDone: number;
  nbFailed: number;
  nbScheduled: number;
  nbSkipped: number;
  created_string: string;
}

export class RepositoryHookEventAnalysis {
  analyze_id: string;
  status: string;
  project_key: string;
  /** What the analysis reported, also on a Success that left files out */
  error?: string;
}

export enum HookEventWorkflowStatus {
  Scheduled = "Scheduled",
  Skipped = "Skipped",
  Error = "Error",
  Done = "Done"
}

export class RepositoryHookEventExtractedData {
  ref: string;
  commit: string;
  cds_event_name: WorkflowHookEventName;
  cds_event_type: string;
}

export class RepositoryHookWorkflow {
  project_key: string;
  vcs_identifier: string;
  repository_identifier: string;
  workflow_name: string;
  type: string;
  status: string;
  run_id: string;
  run_number: string;
  error: string;
  /** Set for a scheduler, workflow-run or manual hook: the owner of the workflow, who the run is attributed to. */
  initiator: Initiator;
  operation_status?: number;
  operation_error?: string;
}

export class StartPurgeResponse {
  report_id: string;
}

export class ProjectRunRetention {
  id: string;
  project_key: string;
  last_modified: string;
  retentions: Retention;
  last_execution: string;
  last_status: string;
  last_report: any;
}

export class Retention {
  retention: Array<WorkflowRetentions>;
  default_retention: RetentionRule;
}

export class RetentionRule {
  duration_in_days: number;
  count: number;
}

export class WorkflowRetentions {
  workflow: string
  rules: Array<WorkflowRetentionRule>;
}

export class WorkflowRetentionRule {
  git_ref: string;
  duration_in_days: number;
  count: number;
}