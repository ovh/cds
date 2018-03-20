import {User} from './user.model';
import {Token} from './token.model';

export const adminGroupName = 'shared.infra';

export class Group {
    id: number;
    name: string;
    admins: Array<User>;
    users: Array<User>;
    tokens: Array<Token>;

    constructor() {
        this.name = '';
        this.admins = [];
        this.users = [];
        this.tokens = [];
    }
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
