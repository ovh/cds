import {Requirement} from './requirement.model';
import {Parameter} from './parameter.model';

export class Action {
    id: number;
    name: string;
    type: string;
    description: string;
    requirements: Array<Requirement>;
    parameters: Array<Parameter>;
    actions: Array<Action>;
    final: boolean;
    last_modified: boolean;
    enabled: boolean;

    // UI parameter
    hasChanged: boolean;
}
