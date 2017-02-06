import {GroupPermission} from '../../model/group.model';

export class PermissionEvent {
    type: string;
    gp: GroupPermission;

    constructor(type: string, gp: GroupPermission) {
        this.type = type;
        this.gp = gp;
    }
}
