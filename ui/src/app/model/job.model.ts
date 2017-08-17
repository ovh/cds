import {Action, ActionWarning} from './action.model';

export class Job {
    pipeline_action_id: number;
    action: Action;
    enabled: boolean;
    last_modified: boolean;
    step_status: Array<StepStatus>;
    warnings: Array<ActionWarning>

    // UI parameter
    hasChanged: boolean;

    constructor() {
        this.action = new Action();
    }
}

export class StepStatus {
    step_order: number;
    status: string;
}
