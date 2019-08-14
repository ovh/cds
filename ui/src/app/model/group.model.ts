import { WithKey } from 'app/shared/table/data-table.component';
import { User } from './user.model';

export const SharedInfraGroupName = 'shared.infra';

export class Group implements WithKey {
    id: number;
    name: string;
    members: Array<User>;
    admin: boolean;

    constructor() {
        this.name = '';
        this.members = [];
    }

    key(): string {
        return `${this.id}`;
    }
}

export class GroupMember {
    id: string;
    username: string;
    fullname: string;
    admin: boolean;
}

export class Groups {
    groups: Array<Group>;
    groups_admin: Array<Group>;

    constructor() {
        this.groups = [];
        this.groups_admin = [];
    }
}

export class GroupPermission {
    group: Group;
    permission: number;

    // flag to know if permission has changed
    hasChanged = false;
    updating = true;

    constructor() {
        this.group = new Group();
        this.permission = 4;
    }
}
