import { Environment } from 'app/model/environment.model';
import { GroupPermission } from 'app/model/group.model';

export class PermissionEvent {
    type: string;
    gp: GroupPermission;
    env: Environment;

    constructor(type: string, gp: GroupPermission) {
        this.type = type;
        this.gp = gp;
    }
}

export class EnvironmentPermissionEvent {
    type: string;
    gp: Array<GroupPermission>;
    env: Environment;

    constructor(type: string, env: Environment, gp: Array<GroupPermission>) {
        this.type = type;
        this.gp = gp;
        this.env = env;
    }
}

