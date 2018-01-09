// WorkflowRun is an execution instance of a run
import {Workflow} from './workflow.model';
import {Stage} from './stage.model';
import {Parameter} from './parameter.model';
import {SpawnInfo, Tests} from './pipeline.model';
import {Commit} from './repositories.model';
import {Job} from './job.model';
import {Hatchery} from './hatchery.model';
import {User} from './user.model';


export class RunNumber {
    num: number;
}
export class WorkflowRunRequest {
    hook: WorkflowNodeRunHookEvent;
    manual: WorkflowNodeRunManual;
    number: number;
    from_nodes: Array<number>;
}

export class WorkflowRun {
    id: number;
    num: number;
    last_subnumber: number;
    project_id: number;
    workflow_id: number;
    workflow: Workflow;
    start: string;
    status: string;
    last_modified: string;
    last_execution: string;
    nodes: { [key: string]: Array<WorkflowNodeRun>; };
    tags: Array<WorkflowRunTags>;
    join_triggers_run: Map<number, TriggerRun>;
    commits: Array<Commit>;
    infos: Array<SpawnInfo>;
}

export class WorkflowRunTags {
    tag: string;
    value: string;
}

// WorkflowNodeRun is as execution instance of a node
export class WorkflowNodeRun {
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
    manual: WorkflowNodeRunManual;
    source_node_runs: Array<number>;
    payload: {};
    pipeline_parameters: Array<Parameter>;
    build_parameters: Array<Parameter>;
    artifacts: Array<WorkflowNodeRunArtifact>;
    tests: Tests;
    commits: Array<Commit>;
    triggers_run: Map<number, TriggerRun>
    can_be_run: boolean
}

export class TriggerRun {
    workflow_dest_node_id: number;
    status: string;
}
// WorkflowNodeRunArtifact represents tests list
export class WorkflowNodeRunArtifact {
    workflow_id: number;
    workflow_node_run_id: number;
    id: number;
    name: string;
    tag: string;
    download_hash: string;
    size: number;
    perm: number;
    md5sum: string;
    object_path: string;
    created: string;
}

// WorkflowNodeJobRun represents an job to be run
export class WorkflowNodeJobRun {
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
    bookedby: Hatchery;
    spawninfos: Array<SpawnInfo>;
}

// WorkflowNodeRunHookEvent is an instanc of event received on a hook
export class WorkflowNodeRunHookEvent {
    payload: {};
    pipeline_parameter: Array<Parameter>;
    workflow_node_hook_id: number;
}

// WorkflowNodeRunManual is an instanc of event received on a hook
export class WorkflowNodeRunManual {
    payload: {};
    pipeline_parameter: Array<Parameter>;
    user: User;
}
