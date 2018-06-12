// WorkflowRun is an execution instance of a run
import {Workflow} from './workflow.model';
import {Stage} from './stage.model';
import {Parameter} from './parameter.model';
import {SpawnInfo, Tests} from './pipeline.model';
import {Commit} from './repositories.model';
import {Job, StepStatus} from './job.model';
import {Hatchery} from './hatchery.model';
import {User} from './user.model';
import {Event} from './event.model';
import {Action} from './action.model';


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
    triggers_run: Map<string, TriggerRun>;
    can_be_run: boolean;

    static fromEventRunWorkflowNode(e: Event): WorkflowNodeRun {
        let wnr = new WorkflowNodeRun();
        wnr.id = e.payload['ID'];
        wnr.subnumber = e.payload['SubNumber'];
        wnr.num = e.payload['Number'];
        wnr.status = e.payload['Status'];
        wnr.start = new Date(e.payload['Start'] * 1000).toString();
        wnr.done = new Date(e.payload['Done'] * 1000).toString();
        wnr.manual = e.payload['Manual'];
        wnr.workflow_node_id = e.payload['NodeID'];
        wnr.workflow_run_id = e.payload['RunID'];
        wnr.stages = new Array<Stage>();
        e.payload['StagesSummary'].forEach(s => {
            let stage = new Stage();
            stage.id = s['ID'];
            stage.build_order = s['BuildOrder'];
            stage.enabled = s['Enabled'];
            stage.name = s['Name'];
            stage.status = s['Status'];
            stage.run_jobs = new Array<WorkflowNodeJobRun>();
            if (s['RunJobsSummary']) {
                s['RunJobsSummary'].forEach(rjs => {
                    let wnjr = new WorkflowNodeJobRun();
                    if (rjs['Done'] > 0) {
                        wnjr.done = new Date(rjs['Done'] * 1000).toString();
                    }
                    wnjr.id = rjs['ID'];
                    if (rjs['Queued'] > 0) {
                        wnjr.queued = new Date(rjs['Queued'] * 1000).toString();
                    }
                    if (rjs['Start'] > 0) {
                        wnjr.start = new Date(rjs['Start'] * 1000).toString();
                    }
                    wnjr.status = rjs['Status'];
                    wnjr.workflow_node_run_id = rjs['WorkflowNodeRunID'];
                    wnjr.job = new Job();
                    wnjr.job.step_status = new Array<StepStatus>();
                    if (rjs['StepStatusSummary']) {
                        rjs['StepStatusSummary'].forEach(sss => {
                           let ss = new StepStatus();
                           if (sss['Done'] > 0) {
                               wnjr.done = new Date(sss['Done'] * 1000).toString();
                           }
                            if (sss['Start'] > 0) {
                                wnjr.start = new Date(sss['Start'] * 1000).toString();
                            }
                           ss.status = sss['Status'];
                           ss.step_order = sss['StepOrder'];
                            wnjr.job.step_status.push(ss);
                        });
                    }

                    wnjr.job.action = new Action();
                    let eventJob = rjs['Job'];
                    wnjr.job.pipeline_stage_id = eventJob['PipelineStageID'];
                    wnjr.job.pipeline_action_id = eventJob['PipelineActionID'];
                    wnjr.job.action.name = eventJob['JobName'];
                    wnjr.job.action.actions = new Array<Action>();
                    if (eventJob['Steps']) {
                        eventJob['Steps'].forEach(step => {
                           let jobStep = new Action();
                           jobStep.name = step['Name'];
                            wnjr.job.action.actions.push(jobStep);
                        });
                    }

                    stage.run_jobs.push(wnjr);
                });
            }

            stage.jobs = new Array<Job>();
            if (s['Jobs']) {
                s['Jobs'].forEach(j => {
                   let job = new Job();
                   job.enabled = j['Enabled'];
                   job.pipeline_action_id = j['PipelineActionID'];
                   job.pipeline_stage_id = j['PipelineStageID'];
                   job.last_modified = (new Date(j['LastModified'] * 1000)).toString();
                   job.action = new Action();
                   let eventAction = j['Action'];
                   job.action.enabled = eventAction['Enabled'];
                   job.action.name = eventAction['Name'];
                   job.action.description = eventAction['Description'];
                   job.action.actions = new Array<Action>();
                   if (eventAction['Actions']) {
                       eventAction['Actions'].forEach(actionStep => {
                          let jobStep = new Action();
                          jobStep.name = actionStep['Name'];
                          jobStep.enabled = actionStep['Enabled'];
                          jobStep.id = actionStep['ID'];

                           job.action.actions.push(jobStep);
                       });
                   }

                   stage.jobs.push(job);
                });
            }
            wnr.stages.push(stage);
        });
        return wnr;
    }
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
    sha512sum: string;
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

    // UI infos for queue
    duration: string;
    updating: boolean;
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
