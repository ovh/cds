import {Group} from './group.model';
import {Requirement} from './requirement.model';
import {User} from './user.model';

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
    check_registration: boolean;
    need_registration: boolean;
    last_registration: string;
    group: Group;
    is_official: boolean;
    is_deprecated: boolean;
    pattern_name: string;

    constructor() {
      this.model_docker = new ModelDocker();
      this.model_virtual_machine = new ModelVirtualMachine();
    }
}

export class ModelDocker {
  image: string;
  shell: string;
  envs: {};
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
    shell?: string;
    envs?: {};
    pre_cmd?: string;
    cmd: string;
    post_cmd?: string;
  };

  constructor() {
    this.model = {
      cmd: ''
    };
  }
}
