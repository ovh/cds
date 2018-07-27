import {WorkflowItem} from './application.workflow.model';
import {GroupPermission} from './group.model';
import {Hook} from './hook.model';
import {Key} from './keys.model';
import {Notification} from './notification.model';
import {Parameter} from './parameter.model';
import {Pipeline, PipelineBuild} from './pipeline.model';
import {RepositoryPoller} from './polling.model';
import {Scheduler} from './scheduler.model';
import {Usage} from './usage.model';
import {Variable} from './variable.model';
import {VCSStrategy} from './vcs.model';

export const applicationNamePattern: RegExp = new RegExp('^[a-zA-Z0-9._-]{1,}$');

export class Application {
    id: number;
    name: string;
    description: string;
    icon: string;
    groups: Array<GroupPermission>;
    variables: Array<Variable>;
    pipelines: Array<ApplicationPipeline>;
    pipelines_build: Array<PipelineBuild>;
    permission: number;
    notifications: Array<Notification>;
    last_modified: string;
    vcs_server: string;
    repository_fullname: string;
    pollers: Array<RepositoryPoller>;
    hooks: Array<Hook>;
    workflows: Array<WorkflowItem>;
    schedulers: Array<Scheduler>;
    workflow_migration: string;
    usage: Usage;
    keys: Array<Key>;
    vcs_strategy: VCSStrategy;
    deployment_strategies: Map<string, any>;
    project_key: string; // project unique key

    // true if someone has updated the application ( used for warnings )
    externalChange: boolean;

    // workflow depth for horizontal tree view
    horizontalDepth: number;

    // Return true if pattern is good
    public static checkName(name: string): boolean {
      if (!name) {
          return false;
      }

      if (!applicationNamePattern.test(name)) {
          return false;
      }
      return true;
    }
}

export class ApplicationPipeline {
    id: number;
    pipeline: Pipeline;
    parameters: Array<Parameter>;
    last_modified: number;
}

export interface ApplicationFilter {
    remote: string;
    branch: string;
    version: string;
}

export class Vulnerability {
    id: number;
    application_id: number;
    workflow_id: number;
    workflow_node_run_id: number;
    title: string;
    description: string;
    cve: string;
    link: string;
    component: string;
    version: string;
    origin: string;
    severity: string;
    fix_in: string;
    ignored: boolean;
}

export class Severity {
    static UNKNOWN = 'unknown';
    static NEGLIGIBLE = 'negligible';
    static LOW = 'low';
    static MEDIUM = 'medium';
    static HIGH = 'high';
    static CRITICAL = 'critical';
    static DEFCON1 = 'defcon1';
}
