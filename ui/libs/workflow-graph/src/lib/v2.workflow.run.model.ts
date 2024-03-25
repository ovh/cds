export class V2WorkflowRun {
    id: string;
    project_key: string;
    vcs_server_id: string;
    repository_id: string;
    workflow_name: string;
    workflow_sha: string;
    workflow_ref: string;
    status: string;
    run_number: number;
    run_attempt: number;
    started: string;
    last_modified: string;
    to_delete: boolean;
    workflow_data: WorkflowData;
    user_id: string;
    username: string;
    contexts: any;
    event: WorkflowEvent;
    job_events: V2WorkflowRunJobEvent[];
    results: Array<WorkflowRunResult>;
}
 export class V2WorkflowRunJobEvent {
    username: string;
    job_id: string;
    inputs: {[key:string]:any};
    run_attempt: number;
 }

export class WorkflowEvent {
    hook_type: string
    ref: string;
    event_name: string;
    sha: string;
    payload: string;
    entity_updated: string;
}

export class WorkflowData {
    workflow: any;
    worker_models: {[key:string]: { }};
    actions:  {[key:string]: { }};
}

export class Gate {
    inputs: {[key:string]:GateInput};
    reviewers: GateReviewers;
}

export class GateInput {
    type: string;
    default: any;
    values: string;
}

export class GateReviewers {
    groups: string[];
    users: string[];
}

export class V2WorkflowRunJob {
    id: string;
    job_id: string;
    workflow_run_id: string;
    project_key: string;
    workflow_name: string;
    run_number: number
    run_attempt: number;
    status: string;
    queued: string;
    scheduled: string;
    started: string;
    ended: string;
    job: {};
    worker_id: string;
    worker_name: string;
    hatchery_name: string;
    outputs: {[key:string]:string};
    steps_status: {[key:string]:StepStatus };
    user_id: string;
    username: string;
    region: string;
    model_type: string;
    matrix: {[key:string]: string};
}

export class StepStatus {
    conclusion: string;
    outcome: string;
    outputs: {[key:string]:string};
    started: string;
    ended: string;
}

export class WorkflowRunInfo {
    id: string;
    workflow_run_id: string;
    issued_at: string;
    level: string;
    message: string;
}

export class WorkflowRunResult {
    id: string;
    type: string;
    detail: WorkflowRunResultDetail;
}

export class WorkflowRunResultDetail {
    Data: any;
    Type: string;
}
