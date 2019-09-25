import {Action, ActionWarning} from './action.model';

export class Job {
    pipeline_stage_id: number;
    pipeline_action_id: number;
    action: Action;
    enabled: boolean;
    last_modified: string;
    step_status: Array<StepStatus>;
    warnings: Array<ActionWarning>;
    worker_name: string;
    worker_id: string;

    // UI parameter
    hasChanged: boolean;
    ref: number;

    constructor() {
        this.action = new Action();
        this.action.enabled = true;
        this.ref = new Date().getTime();
    }
}

export class StepStatus {
    step_order: number;
    status: string;
    start: string;
    done: string;
}
