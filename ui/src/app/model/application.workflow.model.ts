import {Project} from './project.model';
import {Application} from './application.model';
import {Environment} from './environment.model';
import {Pipeline} from './pipeline.model';
import {Trigger} from './trigger.model';

export class WorkflowItem {
    // API Data
    project: Project;
    application: Application;
    environment: Environment;
    pipeline: Pipeline;
    subPipelines: Array<WorkflowItem>;
    trigger: Trigger;

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
};
