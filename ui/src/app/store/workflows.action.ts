import { GroupPermission } from 'app/model/group.model';
import { Label } from 'app/model/project.model';
import { WNode, WNodeHook, WNodeTrigger, Workflow, WorkflowNotification } from 'app/model/workflow.model';

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

export class UpdateWorkflowIcon {
    static readonly type = '[Workflow] Update Workflow Icon';
    constructor(public payload: { projectKey: string, workflowName: string, icon: string }) { }
}

export class DeleteWorkflowIcon {
    static readonly type = '[Workflow] Delete Workflow Icon';
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

//  ------- Group Permission --------- //
export class AddGroupInAllWorkflows {
    static readonly type = '[Workflow] Add Group in Workflows already cached';
    constructor(public payload: { projectKey: string, group: GroupPermission }) { }
}
export class AddGroupInWorkflow {
    static readonly type = '[Workflow] Add Group in Workflow';
    constructor(public payload: { projectKey: string, workflowName: string, group: GroupPermission }) { }
}
export class UpdateGroupInWorkflow {
    static readonly type = '[Workflow] Update Group in Workflow';
    constructor(public payload: { projectKey: string, workflowName: string, group: GroupPermission }) { }
}
export class DeleteGroupInWorkflow {
    static readonly type = '[Workflow] Delete Group in Workflow';
    constructor(public payload: { projectKey: string, workflowName: string, group: GroupPermission }) { }
}

//  ------- Nodes --------- //
export class AddNodeTriggerWorkflow {
    static readonly type = '[Workflow] Add Node Trigger in Workflow';
    constructor(public payload: {
        projectKey: string, workflowName: string, parentId: number, trigger: WNodeTrigger
    }) { }
}

//  ------- Joins --------- //
export class AddJoinWorkflow {
    static readonly type = '[Workflow] Add Join in Workflow';
    constructor(public payload: {
        projectKey: string, workflowName: string, join: WNode
    }) { }
}

//  ------- Hooks --------- //
export class AddHookWorkflow {
    static readonly type = '[Workflow] Add Hook in Workflow';
    constructor(public payload: {
        projectKey: string, workflowName: string, hook: WNodeHook
    }) { }
}
export class UpdateHookWorkflow {
    static readonly type = '[Workflow] Update Hook in Workflow';
    constructor(public payload: {
        projectKey: string, workflowName: string, hook: WNodeHook
    }) { }
}
export class DeleteHookWorkflow {
    static readonly type = '[Workflow] Delete Hook in Workflow';
    constructor(public payload: {
        projectKey: string, workflowName: string, hook: WNodeHook
    }) { }
}

//  ------- Labels --------- //
export class LinkLabelOnWorkflow {
    static readonly type = '[Workflow] Link Label on Workflow';
    constructor(public payload: { projectKey: string, workflowName: string, label: Label }) { }
}
export class UnlinkLabelOnWorkflow {
    static readonly type = '[Workflow] Unlink Label on Workflow';
    constructor(public payload: { projectKey: string, workflowName: string, label: Label }) { }
}

//  ------- Notifications --------- //
export class AddNotificationWorkflow {
    static readonly type = '[Workflow] Add Notification in Workflow';
    constructor(public payload: { projectKey: string, workflowName: string, notification: WorkflowNotification }) { }
}
export class UpdateNotificationWorkflow {
    static readonly type = '[Workflow] Update Notification in Workflow';
    constructor(public payload: { projectKey: string, workflowName: string, notification: WorkflowNotification }) { }
}
export class DeleteNotificationWorkflow {
    static readonly type = '[Workflow] Delete Notification in Workflow';
    constructor(public payload: { projectKey: string, workflowName: string, notification: WorkflowNotification }) { }
}

//  ------- Misc --------- //
export class FetchAsCodeWorkflow {
    static readonly type = '[Workflow] Fetch Workflow As Code';
    constructor(public payload: { projectKey: string, workflowName: string }) { }
}

export class PreviewWorkflow {
    static readonly type = '[Workflow] Preview Workflow';
    constructor(public payload: { projectKey: string, workflowName: string, wfCode: string }) { }
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

export class UpdateFavoriteWorkflow {
    static readonly type = '[Workflow] Update Workflow Favorite';
    constructor(public payload: { projectKey: string, workflowName: string }) { }
}

export class ClearCacheWorkflow {
    static readonly type = '[Workflow] Clear cache Workflow';
    constructor() { }
}

