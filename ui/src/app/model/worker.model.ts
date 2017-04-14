import {Requirement} from './requirement.model';
import {User} from './user.model';

export class WorkerModel {
    id: number;
    name: string;
    type: string;
    image: string;
    capabilities: Array<Requirement>;
    created_by: User;
    owner_id: number;
    group_id: number;
}
