import { Key } from './keys.model';
import { Usage } from './usage.model';
import { Variable } from './variable.model';

export class Environment {
    id: number;
    name: string;
    variables: Array<Variable>;
    keys: Array<Key>;
    permission: number;
    last_modified: number;
    usage: Usage;
    from_repository: string;

    mute: boolean;
}
