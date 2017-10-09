import {Workflow} from './workflow.model';
import {Environment} from './environment.model';
import {Application} from './application.model';
import {Pipeline} from './pipeline.model';

export class Usage {
    workflows: Array<Workflow>;
    environments: Array<Environment>;
    applications: Array<Application>;
    pipelines: Array<Pipeline>;

    constructor() {
        this.applications = new Array<Application>();
        this.workflows = new Array<Workflow>();
        this.pipelines = new Array<Pipeline>();
        this.environments = new Array<Environment>();
    }
}
