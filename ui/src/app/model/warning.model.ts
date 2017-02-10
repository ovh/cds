import {Project} from './project.model';
import {Action} from './action.model';
import {Application} from './application.model';
import {Environment} from './environment.model';
import {Pipeline} from './pipeline.model';

export class WarningsUI {
    [key: string]: WarningUI;
}

export class WarningUI {
    pipelines: WarningsPipeline;
    applications: WarningsApplication;
    environments: WarningsEnvironment; // Not implemented
    variables: WarningAPI[]; // Not implemented

    constructor() {
        this.pipelines = new WarningsPipeline();
        this.applications = new WarningsApplication();
        this.environments = new WarningsEnvironment();
    }
}

export class WarningsPipeline {
    [name: string]: WarningPipeline;
}

export class WarningPipeline {
    parameters: WarningAPI[]; // Not implemented
    jobs: WarningAPI[];

    constructor() {
        this.parameters = [];
        this.jobs = [];
    }
}

export class WarningsApplication {
    [name: string]: WarningApplication;
}

export class WarningApplication {
    variables: WarningAPI[]; // Not implemented
    actions: WarningAPI[];

    constructor() {
        this.variables = [];
        this.actions = [];
    }
}

export class WarningsEnvironment {
    [name: string]: WarningEnvironment;
}

export class WarningEnvironment {
    variables: WarningAPI[]; // Not implemented

    constructor() {
        this.variables = [];
    }
}

export class WarningAPI {
    action: Action;
    application: Application;
    environment: Environment;
    id: number;
    message: string;
    message_param:  MessageParams;
    pipeline: Pipeline;
    project: Project;
    stage_id: number;
}

export interface MessageParams {
    [key: string]: string;
};
