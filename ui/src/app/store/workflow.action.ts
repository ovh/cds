import { GroupPermission } from 'app/model/group.model';
import { ProjectIntegration } from 'app/model/integration.model';
import { RetentionDryRunEvent } from 'app/model/purge.model';
import { WNode, WNodeHook, WNodeTrigger, Workflow, WorkflowNotification } from 'app/model/workflow.model';
import { WorkflowNodeRun, WorkflowRun, WorkflowRunSummary } from 'app/model/workflow.run.model';

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

// ---------  Workflow Run ---
export class ChangeToRunView {
    static readonly type = '[Workflow] Change to Run View';
    constructor(public payload: {}) { }
}

export class GetWorkflowRun {
    static readonly type = '[Workflow] Get Workflow Run';
    constructor(public payload: { projectKey: string, workflowName: string, num: number }) { }
}

export class RemoveWorkflowRunFromList {
    static readonly type = '[Workflow] Remove Workflow Run From List';
    constructor(public payload: { projectKey: string, workflowName: string, num: number }) { }
}

export class SetWorkflowRuns {
    static readonly type = '[Workflow] Set Workflow Runs';
    constructor(public payload: { projectKey: string, workflowName: string, runs: Array<WorkflowRunSummary>, filters?: {}}) { }
}

export class DeleteWorkflowRun {
    static readonly type = '[Workflow] Delete Workflow Run';
    constructor(public payload: { projectKey: string, workflowName: string, num: number }) { }
}

export class CleanWorkflowRun {
    static readonly type = '[Workflow] Clean Workflow Run';
    constructor(public payload: {}) { }
}

export class ClearListRuns {
    static readonly type = '[Workflow] Clear List Workflow Run';
    constructor() { }
}

export class GetWorkflowNodeRun {
    static readonly type = '[Workflow] Get Workflow Node Run';
    constructor(public payload: { projectKey: string, workflowName: string, num: number, nodeRunID: number }) { }
}

export class SelectWorkflowNode {
    static readonly type = '[Workflow] Select Workflow Node';
    constructor(public payload: { node: WNode }) { }
}

export class SelectWorkflowNodeRun {
    static readonly type = '[Workflow] Select Workflow Node Run';
    constructor(public payload: { workflowNodeRun: WorkflowNodeRun, node: WNode }) { }
}

export class SelectWorkflowNodeRunJob {
    static readonly type = '[Workflow] Select Workflow Node Job Run';
    constructor(public payload: { jobID: number}) { }
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

//  ------- Event Integrations --------- //
export class UpdateEventIntegrationsWorkflow {
    static readonly type = '[Workflow] Update Event Integration in Workflow';
    constructor(public payload: { projectKey: string, workflowName: string, eventIntegrations: ProjectIntegration[] }) { }
}
export class DeleteEventIntegrationWorkflow {
    static readonly type = '[Workflow] Delete Event Integration in Workflow';
    constructor(public payload: { projectKey: string, workflowName: string, integrationId: number }) { }
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

export class CancelWorkflowEditMode {
    static readonly type = '[Workflow] Cancel workflow edit modal';
    constructor() { }
}

export class CleanRetentionDryRun {
    static readonly type = '[Workflow] Clean retention dry run';
    constructor() { }
}

export class ComputeRetentionDryRunEvent {
    static readonly type = '[Workflow] Retention dry run event';
    constructor(public payload: { projectKey: string, workflowName: string, event: RetentionDryRunEvent }) { }
}
