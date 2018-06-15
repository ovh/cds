import {Application} from './application.model';
import {Environment} from './environment.model';
import {Parameter} from './parameter.model';
import {Pipeline} from './pipeline.model';
import {Prerequisite} from './prerequisite.model';
import {Project} from './project.model';

export class Trigger {
    id: number;
    src_project: Project;
    src_application: Application;
    src_pipeline: Pipeline;
    src_environment: Environment;
    dest_project: Project;
    dest_application: Application;
    dest_pipeline: Pipeline;
    dest_environment: Environment;
    manual: boolean;
    parameters: Array<Parameter>;
    prerequisites: Array<Prerequisite>;
    last_modified: number;

    // flag to know if variable data has changed
    hasChanged: boolean;
    updating: boolean;

    constructor() {
        this.id = 0;
        this.dest_application = new Application();
        this.dest_pipeline = new Pipeline();
        this.dest_environment = new Environment();
    }
}
