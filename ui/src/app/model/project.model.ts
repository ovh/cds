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
    applications: Array<Application>;
    groups: Array<GroupPermission>;
    variables: Array<Variable>;
    environments: Array<Environment>;
    repositories_manager: Array<RepositoriesManager>;
    permission: number;
    last_modified: string;
    workflow_migration: string;

    // true if someone has updated the project ( used for warnings )
    externalChange: boolean;
}
