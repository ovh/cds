import { Requirement } from './requirement.model';
import { User } from './user.model';

export class Hatchery {
    id: number;
    uid: string;
    name: string;
    status: string;
    group_id: number;
    model: Model;
}

export class Model {
    id: number;
    name: string;
    type: string;
    image: string;
    capabilities: Array<Requirement>;
    communication: string;
    template: {};
    run_script: string;
    disabled: boolean;
    need_registration: boolean;
    last_registration: string;
    user_last_modified: string;
    created_by: User;
    group_id: number;
}
