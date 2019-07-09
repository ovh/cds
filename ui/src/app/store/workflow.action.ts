import { GroupPermission } from 'app/model/group.model';
import { WNode, WNodeHook, WNodeTrigger, Workflow, WorkflowNotification } from 'app/model/workflow.model';
import { WorkflowNodeRun, WorkflowRun } from 'app/model/workflow.run.model';

// ---------  MODAL  -----

export class OpenEditModal {
    static readonly type = '[Workflow] Open Edit Modal';
    constructor(public payload: { node: WNode, hook: WNodeHook }) { }
}

export class CloseEditModal {
    static readonly type = '[Workflow] Close Edit Modal';
    constructor(public payload: {}) { }
}

export class UpdateModal {
    static readonly type = '[Workflow] UpdateModal';
    constructor(public payload: { workflow: Workflow }) { }
}

// ---------  Sidebar -----
export class SidebarRunsMode {
    static readonly type = '[Workflow] Sidebar run mode';
    constructor(public payload: {}) { }
}

// ---------  Workflow Run ---
export class ChangeToRunView {
    static readonly type = '[Workflow] Change to Run View';
    constructor(public payload: {}) { }
}

export class GetWorkflowRun {
    static readonly type = '[Workflow] Get Workflow Run';
    constructor(public payload: { projectKey: string, workflowName: string, num: number }) { }
}

export class GetWorkflowRuns {
    static readonly type = '[Workflow] Get Workflow Runs';
    constructor(public payload: { projectKey: string, workflowName: string, limit: string }) { }
}

export class DeleteWorkflowRun {
    static readonly type = '[Workflow] Delete Workflow Run';
    constructor(public payload: { projectKey: string, workflowName: string, num: number }) { }
}

export class CleanWorkflowRun {
    static readonly type = '[Workflow] Clean Workflow Run';
    constructor(public payload: {}) { }
}

export class GetWorkflowNodeRun {
    static readonly type = '[Workflow] Get Workflow Node Run';
    constructor(public payload: { projectKey: string, workflowName: string, num: number, nodeRunID: number }) { }
}

export class SelectWorkflowNodeRun {
    static readonly type = '[Workflow] Select Workflow Node Run';
    constructor(public payload: { workflowNodeRun: WorkflowNodeRun, node: WNode }) { }
}

export class UpdateWorkflowRunList {
    static readonly type = '[Workflow] Update Workflow Run List';
    constructor(public payload: { workflowRun: WorkflowRun }) { }
}

// ---------  Workflow  -----

export class CreateWorkflow {
    static readonly type = '[Workflow] Create Workflow';
    constructor(public payload: { projectKey: string, workflow: Workflow }) { }
}

export class ImportWorkflow {
    static readonly type = '[Workflow] Import Workflow';
    constructor(public payload: { projectKey: string, workflowCode: string, wfName?: string, force?: boolean }) { }
}

export class GetWorkflow {
    static readonly type = '[Workflow] Get Workflow';
    constructor(public payload: { projectKey: string, workflowName: string }) { }
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
export class SelectHook {
    static readonly type = '[Workflow] Select hook';
    constructor(public payload: { hook: WNodeHook, node: WNode }) { }
}
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

export class UpdateFavoriteWorkflow {
    static readonly type = '[Workflow] Update Workflow Favorite';
    constructor(public payload: { projectKey: string, workflowName: string }) { }
}

export class CleanWorkflowState {
    static readonly type = '[Workflow] Clean Workflow State';
    constructor() { }
}

