import { Permission } from 'app/model/permission.model';
import { Application } from './application.model';
import { Environment } from './environment.model';
import { GroupPermission } from './group.model';
import { ProjectIntegration } from './integration.model';
import { Key } from './keys.model';
import { Pipeline } from './pipeline.model';
import { RepositoriesManager } from './repositories.model';
import { Variable } from './variable.model';
import { Workflow } from './workflow.model';

export class Project {
  key: string;
  name: string;
  description: string;
  icon: string;
  workflows: Array<Workflow>;
  workflow_names: Array<IdName>;
  pipelines: Array<Pipeline>;
  pipeline_names: Array<IdName>;
  applications: Array<Application>;
  application_names: Array<IdName>;
  groups: Array<GroupPermission>;
  variables: Array<Variable>;
  environments: Array<Environment>;
  environment_names: Array<IdName>;
  permissions: Permission;
  last_modified: string;
  vcs_servers: Array<RepositoriesManager>;
  keys: Array<Key>;
  integrations: Array<ProjectIntegration>;
  features: {};
  labels: Label[];
  metadata: {};
  favorite: boolean;
  // true if someone has updated the project ( used for warnings )
  externalChange: boolean;
  loading: boolean;
  mute: boolean;
  organization: string;
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
  description?: string;
  icon?: string;
  labels?: Label[];
  // ui params
  mute: boolean;
}

export class Label {
  id: number;
  name: string;
  color: string;
  project_id: number;
  workflow_id: number;
  // ui params
  font_color: string;
}

export class VCSProject {
    id: string;
    name: string;
}

export class Repository {
    id: string;
    name: string;
}

export class Entity {
    id: string;
    name: string;
    type: string;
    branch: string;
}
