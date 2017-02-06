import {Requirement} from './requirement.model';
import {Parameter} from './parameter.model';

export class Action {
    id: number;
    name: string;
    type: ActionType;
    description: string;
    requirements: Array<Requirement>;
    parameters: Array<Parameter>;
    actions: Array<Action>;
    final: boolean;
    last_modified: boolean;


    // UI parameter
    hasChanged: boolean;
}

export enum ActionType {
    'Default',
    'Builtin',
    'Plugin',
    'Joined'
}
