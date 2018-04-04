import {Pipeline} from './pipeline.model';
import {Application} from './application.model';
import {GroupPermission} from './group.model';
import {Variable} from './variable.model';
import {Environment} from './environment.model';
import {RepositoriesManager} from './repositories.model';
import {Workflow} from './workflow.model';
import {Key} from './keys.model';
import {ProjectPlatform} from './platform.model';

export class Project {
    key: string;
    name: string;
    workflows: Array<Workflow>;
    workflow_names: Array<string>;
    pipelines: Array<Pipeline>;
    pipeline_names: Array<IdName>;
    applications: Array<Application>;
    application_names: Array<IdName>;
    groups: Array<GroupPermission>;
    variables: Array<Variable>;
    environments: Array<Environment>;
    permission: number;
    last_modified: string;
    workflow_migration: string;
    vcs_servers: Array<RepositoriesManager>;
    keys: Array<Key>;
    platforms: Array<ProjectPlatform>;
    features: {};
    // true if someone has updated the project ( used for warnings )
    externalChange: boolean;
}

export class LoadOpts {
  constructor(
    public queryParam: string,
    public fieldName: string
  ) { }
}

export class IdName {
  id: number;
  name: string;
}
