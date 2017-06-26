import {Requirement} from './requirement.model';
import {Parameter} from './parameter.model';

export class Action {
    id: number;
    name: string;
    type: string;
    description = '';
    requirements: Array<Requirement>;
    parameters: Array<Parameter>;
    actions: Array<Action>;
    final: boolean;
    last_modified: boolean;
    enabled: boolean;

    // UI parameter
    hasChanged: boolean;
    loading: boolean;
}

export class PipelineUsingAction {
    action_id: number;
    type: string;
    action_name: string;
    pipeline_name: string;
    application_name: string;
    project_name: string;
    key: string;
    stage_id: number;
}
