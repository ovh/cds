import {Project} from './project.model';
import {Application} from './application.model';
import {Environment} from './environment.model';
import {Pipeline, PipelineBuild} from './pipeline.model';
import {Trigger} from './trigger.model';
import {Scheduler} from './scheduler.model';
import {RepositoryPoller} from './polling.model';
import {Hook} from './hook.model';

export class WorkflowItem {
    // API Data
    project: Project;
    application: Application;
    environment: Environment;
    pipeline: Pipeline;
    subPipelines: Array<WorkflowItem>;
    trigger: Trigger;
    schedulers: Array<Scheduler>;
    poller: RepositoryPoller;
    hooks: Array<Hook>;

    // Parent data
    parent: ParentItem;
}

export interface ParentItem {
    pipeline_id: number;
    application_id: number;
    environment_id: number;
    buildNumber: number;
    version: number;
    branch: string;
}

export class WorkflowStatusResponse {
    builds: Array<PipelineBuild>;
    schedulers: Array<Scheduler>;
    pollers: Array<RepositoryPoller>;
    hooks: Array<Hook>;
}
