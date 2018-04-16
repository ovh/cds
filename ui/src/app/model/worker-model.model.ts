import {Requirement} from './requirement.model';
import {User} from './user.model';
import {Group} from './group.model';

export class WorkerModel {
    id: number;
    name: string;
    description: string;
    type: string;
    registered_os: string;
    registered_arch: string;
    model_docker: ModelDocker;
    model_virtual_machine: ModelVirtualMachine;
    registered_capabilities: Array<Requirement>;
    created_by: User;
    owner_id: number;
    group_id: number;
    restricted: boolean;
    group: Group;
    is_official: boolean;
    is_deprecated: boolean;

    constructor() {
      this.model_docker = new ModelDocker();
      this.model_virtual_machine = new ModelVirtualMachine();
    }
}

export class ModelDocker {
  image: string;
  cmd: string;
  memory: number;
}

export class ModelVirtualMachine {
  image: string;
  flavor: string;
  pre_cmd: string;
  cmd: string;
  post_cmd: string;
}

export class ModelPattern {
  id: number;
  name: string;
  type: string;
  model: {
    pre_cmd?: string;
    cmd: string;
    post_cmd?: string;
  };
}
