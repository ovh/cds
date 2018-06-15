import {Application} from './application.model';
import {Pipeline} from './pipeline.model';

export class RepositoryPoller {
    name: string;
    application: Application;
    pipeline: Pipeline;
    enabled: boolean;
    date_creation: Date;
    next_execution: RepositoryPollerExecution;


    // Ui params
    updating: boolean;
    hasChanged: boolean;

    constructor() {
        this.name = '';
        this.enabled = true;
    }
}

export class RepositoryPollerExecution {
    execution_planned_date: Date;
}
