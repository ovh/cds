// WorkflowRun is an execution instance of a run
import { WithKey } from 'app/shared/table/data-table.component';
import { Vulnerability } from './application.model';
import { Event } from './event.model';
import { Hatchery } from './hatchery.model';
import { Job } from './job.model';
import { Parameter } from './parameter.model';
import { SpawnInfo, Tests } from './pipeline.model';
import { Commit } from './repositories.model';
import { Stage } from './stage.model';
import { User } from './user.model';
import { WNodeOutgoingHook, Workflow } from './workflow.model';


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
    last_modified_nano: number;
    last_execution: string;
    nodes: { [key: string]: Array<WorkflowNodeRun>; };
    tags: Array<WorkflowRunTags>;
    commits: Array<Commit>;
    infos: Array<SpawnInfo>;
    version: number;

    // Useful for UI
    duration: string;
    force_update: boolean;

    static fromEventRunWorkflow(event: Event): WorkflowRun {
        let wr = new WorkflowRun();
        wr.id = event.payload['ID'];
        wr.status = event.payload['Status'];
        wr.num = event.workflow_run_num;
        wr.start = new Date(event.payload['Start'] * 1000).toString();
        wr.last_execution = new Date(event.payload['LastExecution'] * 1000).toString();
        wr.last_modified = new Date(event.payload['LastModified'] * 1000).toString();
        wr.last_modified_nano = event.payload['LastModifiedNano'];
        wr.tags = event.payload['Tags'].map(t => {
            return { tag: t.Tag, value: t.Value }
        });
        return wr;
    }
}

export class WorkflowRunTags {
    tag: string;
    value: string;
}

// WorkflowNodeRun is as execution instance of a node
export class WorkflowNodeRun implements WithKey {
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
    vulnerabilities_report: WorkflowNodeRunVulnerabilityReport;
    can_be_run: boolean;
    uuid: string;
    outgoinghook: WNodeOutgoingHook;
    hook_execution_timestamp: number;
    execution_id: string;
    callback: WorkflowNodeOutgoingHookRunCallback;

    key(): string {
        return `${this.id}-${this.num}.${this.subnumber}`;
    }
}

export class WorkflowNodeOutgoingHookRunCallback {
    workflow_node_outgoing_hook_id: number;
    start: Date;
    done: Date;
    status: string;
    log: string;
    workflow_run_number: number;
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
    sha512sum: string;
    object_path: string;
    created: string;
}

// WorkflowNodeRunStaticFiles represent static files
export class WorkflowNodeRunStaticFiles {
    workflow_node_run_id: number;
    id: number;
    name: string;
    entrypoint: string;
    public_url: string;
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

    // UI infos for queue
    duration: string;
    updating: boolean;
}

// WorkflowNodeRunHookEvent is an instanc of event received on a hook
export class WorkflowNodeRunHookEvent {
    payload: {};
    pipeline_parameter: Array<Parameter>;
    uuid: string;
    parent_workflow: {
        key: string;
        name: string;
        run: string;
    };
}

// WorkflowNodeRunManual is an instanc of event received on a hook
export class WorkflowNodeRunManual {
    payload: {};
    pipeline_parameter: Array<Parameter>;
    user: User;
}

export class WorkflowNodeRunVulnerabilityReport {
    id: number;
    application_id: number;
    workflow_id: number;
    workflow_run_id: number;
    workflow_node_run_id: number;
    num: number;
    branch: string;
    report: WorkflowNodeRunVulnerability;
}

export class WorkflowNodeRunVulnerability {
    vulnerabilities: Array<Vulnerability>;
    summary: { [key: string]: number };
    default_branch_summary: { [key: string]: number };
    previous_run_summary: { [key: string]: number };
}
