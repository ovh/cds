import {Parameter} from './parameter.model';
import {Requirement} from './requirement.model';

export class Action {
    id: number;
    name: string;
    type: string;
    description = '';
    requirements: Array<Requirement>;
    parameters: Array<Parameter>;
    actions: Array<Action>;
    optional: boolean;
    always_executed: boolean;
    last_modified: boolean;
    enabled: boolean;
    deprecated: boolean;

    // UI parameter
    hasChanged: boolean;
    loading: boolean;
    showAddStep: boolean;
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

export class ActionWarning {
  type: string;
  action: Action;
}
