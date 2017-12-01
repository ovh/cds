import {Project} from './project.model';
import {Action} from './action.model';
import {Application} from './application.model';
import {Environment} from './environment.model';
import {Pipeline} from './pipeline.model';

export class WarningUI {
    pipelines: Map<string, WarningPipeline>;
    applications: Map<string, WarningApplication>;
    environments: Map<string, WarningEnvironment>; // Not implemented
    variables: WarningAPI[]; // Not implemented

    constructor() {
        this.pipelines = new Map<string, WarningPipeline>();
        this.applications = new Map<string, WarningApplication>();
        this.environments = new Map<string, WarningEnvironment>();
        this.variables = new Array<WarningAPI>();
    }
}

export class WarningPipeline {
    parameters: WarningAPI[]; // Not implemented
    jobs: WarningAPI[];

    constructor() {
        this.parameters = [];
        this.jobs = [];
    }
}

export class WarningApplication {
    variables: WarningAPI[]; // Not implemented
    actions: WarningAPI[];

    constructor() {
        this.variables = [];
        this.actions = [];
    }
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
}
