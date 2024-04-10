export class V2WorkflowRun {
    id: string;
    project_key: string;
    vcs_server_id: string;
    vcs_server: string;
    repository_id: string;
    repository: string;
    workflow_name: string;
    workflow_sha: string;
    workflow_ref: string;
    status: string;
    run_number: number;
    run_attempt: number;
    started: string;
    last_modified: string;
    to_delete: boolean;
    workflow_data: WorkflowRunData;
    user_id: string;
    username: string;
    contexts: any;
    event: WorkflowEvent;
    job_events: V2WorkflowRunJobEvent[];
}

export class WorkflowRunData {
    workflow: V2Workflow;
    worker_models: { [key: string]: {} };
    actions: { [key: string]: {} };
}

export class V2WorkflowRunJobEvent {
    user_id: string;
    username: string;
    job_id: string;
    inputs: { [key: string]: any };
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

export class V2Workflow {
    name: string;
    repository: WorkflowRepository;
    'commit-status': CommitStatus;
    on: WorkflowOn;
    stages: { [key: string]: any };
    gates: { [key: string]: V2JobGate };
    jobs: { [key: string]: V2Job };
    env: { [key: string]: string };
    integrations: Array<string>;
    vars: Array<string>;
}

export class WorkflowRepository {
    vcs: string;
    name: string;
}

export class CommitStatus {
    title: string;
    description: string;
}

export class WorkflowOn {
    push: {
        branches: Array<string>;
        tags: Array<string>;
        paths: Array<string>;
    };
    'pull-request': {
        branches: Array<string>;
        comment: string;
        paths: Array<string>;
        types: Array<string>;
    };
    'pull-request-comment': {
        branches: Array<string>;
        comment: string;
        paths: Array<string>;
        types: Array<string>;
    };
    'model-update': {
        models: Array<string>;
        target_branch: string;
    };
    'workflow-update': {
        target_branch: string;
    };
}

export class V2JobGate {
    if: string;
    inputs: { [key: string]: V2JobGateInput };
    reviewers: V2JobGateReviewers;
}

export class V2JobGateInput {
    type: string;
    default: any;
    values: Array<string>;
}

export class V2JobGateReviewers {
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
    status: V2WorkflowRunJobStatus;
    queued: string;
    scheduled: string;
    started: string;
    ended: string;
    job: V2Job;
    worker_id: string;
    worker_name: string;
    hatchery_name: string;
    steps_status: { [key: string]: StepStatus };
    user_id: string;
    username: string;
    region: string;
    model_type: string;
    matrix: { [key: string]: string };
    gate_inputs: { [key: string]: any };
}

export enum V2WorkflowRunJobStatus {
    Waiting = 'Waiting',
    Building = 'Building',
    Fail = 'Fail',
    Stopped = 'Stopped',
    Success = 'Success',
    Scheduling = 'Scheduling',
    Skipped = 'Skipped'
}

export class V2Job {
    name: string;
    if: string;
    gate: string;
    inputs: { [key: string]: string };
    steps: Array<any>;
    needs: Array<string>;
    stage: string;
    region: string;
    'continue-on-error': boolean;
    'runs-on': string;
    strategy: V2JobStrategy;
    integrations: Array<string>;
    vars: Array<string>;
    env: { [key: string]: string };
    services: { [key: string]: any };
}

export class V2JobStrategy {
    matrix: { [key: string]: Array<string> };
}

export class StepStatus {
    conclusion: string;
    outcome: string;
    outputs: { [key: string]: string };
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
    type: WorkflowRunResultType;
    detail: WorkflowRunResultDetail;
}

export enum WorkflowRunResultType {
    tests = 'tests'
}

export class WorkflowRunResultDetail {
    data: any;
    type: string;
}