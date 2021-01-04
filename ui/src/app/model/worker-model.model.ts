import { Group } from './group.model';
import { Requirement } from './requirement.model';
import { User } from './user.model';

export class WorkerModel {
    id: number;
    name: string;
    description: string;
    type: string;
    disabled: boolean;
    restricted: boolean;
    registered_os: string;
    registered_arch: string;
    need_registration: boolean;
    last_registration: string;
    check_registration: boolean;
    user_last_modified: string;
    created_by: User;
    group_id: number;
    nb_spawn_err: number;
    last_spawn_err: string;
    nb_spawn_err_log: string;
    last_spawn_err_log: string;
    date_last_spawn_err: string;
    is_deprecated: boolean;
    model_virtual_machine: ModelVirtualMachine;
    model_docker: ModelDocker;
    editable: boolean;
    group: Group;
    registered_capabilities: Array<Requirement>;
    is_official: boolean;
    pattern_name: string;

    constructor() {
        this.model_docker = new ModelDocker();
        this.model_virtual_machine = new ModelVirtualMachine();
    }
}

export class ModelDocker {
    image: string;
    private: boolean;
    registry: string;
    username: string;
    password: string;
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
    user: string;
    password: string;
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
