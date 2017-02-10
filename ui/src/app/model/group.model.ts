import {User} from './user.model';

export class Group {
    id: number;
    name: string;
    admins: Array<User>;
    users: Array<User>;

    constructor() {
        this.name = '';
        this.admins = [];
        this.users = [];
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
