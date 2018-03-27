import {Requirement} from './requirement.model';
import {User} from './user.model';
import {Group} from './group.model';

export class WorkerModel {
    id: number;
    name: string;
    description: string;
    type: string;
    image: string;
    capabilities: Array<Requirement>;
    created_by: User;
    owner_id: number;
    group_id: number;
    restricted: boolean;
    group: Group;
    is_official: boolean;
    is_deprecated: boolean;
}
