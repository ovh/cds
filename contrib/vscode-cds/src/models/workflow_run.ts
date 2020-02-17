import { Workflow } from "./workflow";

export interface WorkflowRun {
    id: number;
    num: number;
    last_subnumber: number;
    project_id: number;
    workflow_id: number;
    workflow: Workflow;
    start: string;
    status: string;
    last_modified: string;
    last_modified_nano: number;
    last_execution: string;
    nodes: { [key: string]: Array<WorkflowNodeRun>; };
    tags: Array<WorkflowRunTags>;
    // commits: Array<Commit>;
    // infos: Array<SpawnInfo>;
    version: number;
}

export interface WorkflowRunTags {
    tag: string;
    value: string;
}

// WorkflowNodeRun is as execution instance of a node
export interface WorkflowNodeRun {
    workflow_node_name: string,
    workflow_run_id: number;
    id: number;
    workflow_node_id: number;
    num: number;
    subnumber: number;
    status: string;
    stages: Array<Stage>;
    start: string;
    last_modified: string;
    done: string;
    hook_event: WorkflowNodeRunHookEvent;
    //manual: WorkflowNodeRunManual;
    source_node_runs: Array<number>;
    payload: {};
    // pipeline_parameters: Array<Parameter>;
    // build_parameters: Array<Parameter>;
    artifacts: Array<WorkflowNodeRunArtifact>;
    // tests: Tests;
    // commits: Array<Commit>;
    // vulnerabilities_report: WorkflowNodeRunVulnerabilityReport;
    can_be_run: boolean;
    uuid: string;
    //outgoinghook: WNodeOutgoingHook;
    hook_execution_timestamp: number;
    execution_id: string;
    callback: WorkflowNodeOutgoingHookRunCallback;
}

export interface WorkflowNodeOutgoingHookRunCallback {
    workflow_node_outgoing_hook_id: number;
    start: Date;
    done: Date;
    status: string;
    log: string;
    workflow_run_number: number;
}

// WorkflowNodeRunArtifact represents tests list
export interface WorkflowNodeRunArtifact {
    workflow_id: number;
    workflow_node_run_id: number;
    id: number;
    name: string;
    tag: string;
    download_hash: string;
    size: number;
    perm: number;
    md5sum: string;
    sha512sum: string;
    object_path: string;
    created: string;
}

// WorkflowNodeRunStaticFiles represent static files
export interface WorkflowNodeRunStaticFiles {
    workflow_node_run_id: number;
    id: number;
    name: string;
    entrypoint: string;
    public_url: string;
    created: string;
}

// WorkflowNodeJobRun represents an job to be run
export interface WorkflowNodeJobRun {
    id: number;
    workflow_node_run_id: number;
    job: Job;
    parameters: Array<Parameter>;
    status: string;
    queued: string;
    queued_seconds: number;
    start: string;
    done: string;
    model: string;
    //bookedby: Hatchery;
    //spawninfos: Array<SpawnInfo>;

    // UI infos for queue
    duration: string;
    updating: boolean;
}

// WorkflowNodeRunHookEvent is an instanc of event received on a hook
export interface WorkflowNodeRunHookEvent {
    payload: {};
    //pipeline_parameter: Array<Parameter>;
    uuid: string;
    parent_workflow: {
        key: string;
        name: string;
        run: string;
    };
}

export interface Parameter {
    id: number;
    name: string;
    type: string;
    value: string;
    description: string;
    advanced: boolean;

    // flag to know if variable data has changed
    hasChanged: boolean;
    updating: boolean;
    previousName: string;

    // Useful for list on UI
    ref: Parameter;
}

export interface Stage {
    id: number;
    name: string;
    status: string;
    build_order: number;
    enabled: boolean;
    jobs: Array<Job>;
    run_jobs: Array<WorkflowNodeJobRun>;
//    prerequisites: Array<Prerequisite>;
    last_modified: number;
//    warnings: Array<ActionWarning>;
    // UI params
    hasChanged: boolean;
    edit: boolean;
  }

  // WorkflowNodeJobRun represents an job to be run
export interface WorkflowNodeJobRun {
    id: number;
    workflow_node_run_id: number;
    job: Job;
    parameters: Array<Parameter>;
    status: string;
    queued: string;
    queued_seconds: number;
    start: string;
    done: string;
    model: string;
    // bookedby: Hatchery;
    // spawninfos: Array<SpawnInfo>;

    // UI infos for queue
    duration: string;
    updating: boolean;
}

export interface Job {
    pipeline_stage_id: number;
    pipeline_action_id: number;
    action: Action;
    enabled: boolean;
    last_modified: string;
    step_status: Array<StepStatus>;
//    warnings: Array<ActionWarning>;
    worker_name: string;
    worker_id: string;

    // UI parameter
    hasChanged: boolean;
}

export interface StepStatus {
    step_order: number;
    status: string;
    start: string;
    done: string;
}

export interface Action {
    id: number;
    group_id: number;
    name: string;
    step_name: string;
    type: string;
    description: string;
//    requirements: Array<Requirement>;
    parameters: Array<Parameter>;
    actions: Array<Action>;
    optional: boolean;
    always_executed: boolean;
    enabled: boolean;
    deprecated: boolean;
    // group: Group;
    // first_audit: AuditAction;
    // last_audit: AuditAction;
    editable: boolean;
    import_url: string;

    // UI parameter
    hasChanged: boolean;
    loading: boolean;
    showAddStep: boolean;
}