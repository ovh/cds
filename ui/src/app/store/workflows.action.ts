import { Workflow } from 'app/model/workflow.model';

// Use to load fetched Workflow in our app
export class LoadWorkflow {
    static readonly type = '[Workflow] Load Workflow';
    constructor(public payload: { projectKey: string, workflow: Workflow }) { }
}

// Use to fetch Workflow from backend
export class FetchWorkflow {
    static readonly type = '[Workflow] Fetch Workflow';
    constructor(public payload: { projectKey: string, workflowName: string }) { }
}

export class AddWorkflow {
    static readonly type = '[Workflow] Add Workflow';
    constructor(public payload: { projectKey: string, workflow: Workflow }) { }
}

export class ImportWorkflow {
    static readonly type = '[Workflow] Import Workflow';
    constructor(public payload: { projectKey: string, workflowCode: string, wfName?: string, force?: boolean }) { }
}

export class UpdateWorkflow {
    static readonly type = '[Workflow] Update Workflow';
    constructor(public payload: { projectKey: string, workflowName: string, changes: Workflow }) { }
}

export class DeleteWorkflow {
    static readonly type = '[Workflow] Delete Workflow';
    constructor(public payload: { projectKey: string, workflowName: string }) { }
}

//  ------- Audit ---------
export class FetchWorkflowAudits {
    static readonly type = '[Workflow] Fetch Workflow Audits';
    constructor(public payload: { projectKey: string, workflowName: string }) { }
}

export class RollbackWorkflow {
    static readonly type = '[Workflow] Rollback Workflow';
    constructor(public payload: { projectKey: string, workflowName: string, auditId: number }) { }
}


//  ------- Misc --------- //
export class FetchAsCodeWorkflow {
    static readonly type = '[Workflow] Fetch Workflow As Code';
    constructor(public payload: { projectKey: string, workflowName: string }) { }
}

export class PreviewWorkflow {
    static readonly type = '[Workflow] Preview Workflow';
    constructor(public payload: { projectKey: string, workflowName: string, pipCode: string }) { }
}

export class ExternalChangeWorkflow {
    static readonly type = '[Workflow] External Change Workflow';
    constructor(public payload: { projectKey: string, workflowName: string }) { }
}

export class ResyncWorkflow {
    static readonly type = '[Workflow] Resync Workflow';
    constructor(public payload: { projectKey: string, workflowName: string }) { }
}

export class DeleteFromCacheWorkflow {
    static readonly type = '[Workflow] Delete from cache Workflow';
    constructor(public payload: { projectKey: string, workflowName: string }) { }
}


