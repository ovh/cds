import { WorkflowNodeJobRun } from 'app/model/workflow.run.model';

export class SetJobs {
    static readonly type = '[Queue] Set jobs';
    constructor(public payload: Array<WorkflowNodeJobRun>) { }
}

export class AddOrUpdateJob {
    static readonly type = '[Queue] Add or update job';
    constructor(public payload: WorkflowNodeJobRun) { }
}

export class RemoveJob {
    static readonly type = '[Queue] Remove job';
    constructor(public jobID: number) { }
}

export class SetJobUpdating {
    static readonly type = '[Queue] Set job updating';
    constructor(public jobID: number) { }
}
