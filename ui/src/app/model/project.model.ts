import {Pipeline} from './pipeline.model';
import {Application} from './application.model';
import {GroupPermission} from './group.model';
import {Variable} from './variable.model';
import {Environment} from './environment.model';
import {RepositoriesManager} from './repositories.model';
import {Workflow} from './workflow.model';

export class Project {
    key: string;
    name: string;
    workflows: Array<Workflow>;
    pipelines: Array<Pipeline>;
    pipeline_names: Array<string>;
    applications: Array<Application>;
    application_names: Array<string>;
    groups: Array<GroupPermission>;
    variables: Array<Variable>;
    environments: Array<Environment>;
    permission: number;
    last_modified: string;
    workflow_migration: string;
    vcs_servers: Array<RepositoriesManager>;
    // true if someone has updated the project ( used for warnings )
    externalChange: boolean;
}
