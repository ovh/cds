import {Requirement} from './requirement.model';
import {User} from './user.model';
import {Group} from './group.model';

export class WorkerModel {
    id: number;
    name: string;
    description: string;
    type: string;
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
