import {Requirement} from './requirement.model';
import {User} from './user.model';
import {Group} from './group.model';

export class WorkerModel {
    id: number;
    name: string;
    type: string;
    image: string;
    capabilities: Array<Requirement>;
    created_by: User;
    owner_id: number;
    group_id: number;
    group: Group;
}
