import {GroupPermission} from './group.model';
import {Usage} from './usage.model';
import {Variable} from './variable.model';

export class Environment {
    id: number;
    name: string;
    groups: Array<GroupPermission>;
    variables: Array<Variable>;
    permission: number;
    last_modified: number;
    usage: Usage;

    mute: boolean;
}
