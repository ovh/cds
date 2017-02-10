import {Pipeline} from './pipeline.model';
import {Application} from './application.model';

export class RepositoryPoller {
    name: string;
    application: Application;
    pipeline: Pipeline;
    enabled: boolean;
    date_creation: Date;

    constructor() {
        this.name = '';
        this.enabled = true;
    }
}
