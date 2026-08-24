import { WithKey } from 'app/shared/table/data-table.component';

export const SharedInfraGroupName = 'shared.infra';

export class Group implements WithKey {
    id: number;
    name: string;
    members: Array<GroupMember>;
    admin: boolean;
    organization: string;
    no_active_member: boolean;

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
    disabled: boolean;
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
