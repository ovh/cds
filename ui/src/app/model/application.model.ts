import {GroupPermission} from './group.model';
import {PipelineBuild, Pipeline} from './pipeline.model';
import {Notification} from './notification.model';
import {Variable} from './variable.model';
import {Parameter} from './parameter.model';
import {RepositoryPoller} from './polling.model';
import {Hook} from './hook.model';
import {Usage} from './usage.model';
import {WorkflowItem} from './application.workflow.model';
import {Scheduler} from './scheduler.model';
import {Key} from './keys.model';
import {VCSStrategy} from './vcs.model';

export class Application {
    id: number;
    name: string;
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

    project_key: string; // project unique key

    // true if someone has updated the application ( used for warnings )
    externalChange: boolean;

    // workflow depth for horizontal tree view
    horizontalDepth: number;
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
