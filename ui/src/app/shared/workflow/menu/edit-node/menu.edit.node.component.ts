import { Component, Input, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import {PermissionValue} from 'app/model/permission.model';
import {Project} from 'app/model/project.model';
import {
    WNode, WNodeHook,
    WNodeJoin,
    WNodeTrigger,
    WNodeType,
    Workflow,
    WorkflowPipelineNameImpact
} from 'app/model/workflow.model';
import {WorkflowCoreService} from 'app/service/workflow/workflow.core.service';
import {AutoUnsubscribe} from 'app/shared/decorator/autoUnsubscribe';
import {ToastService} from 'app/shared/toast/ToastService';
import {WorkflowDeleteNodeComponent} from 'app/shared/workflow/modal/delete/workflow.node.delete.component';
import {WorkflowHookModalComponent} from 'app/shared/workflow/modal/hook-modal/hook.modal.component';
import {WorkflowNodeEditModalComponent} from 'app/shared/workflow/modal/node-edit/node.edit.modal.component';
import {WorkflowNodeOutGoingHookEditComponent} from 'app/shared/workflow/modal/outgoinghook-edit/outgoinghook.edit.component';
import {WorkflowTriggerComponent} from 'app/shared/workflow/modal/trigger/workflow.trigger.component';
import { AddHookWorkflow, AddJoinWorkflow, AddNodeTriggerWorkflow, UpdateHookWorkflow, UpdateWorkflow } from 'app/store/workflows.action';
import { cloneDeep } from 'lodash';
import {IPopup} from 'ng2-semantic-ui';
import { ActiveModal } from 'ng2-semantic-ui/dist';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-workflow-menu-wnode-edit',
    templateUrl: './menu.edit.node.html',
    styleUrls: ['./menu.edit.node.scss']
})
@AutoUnsubscribe()
export class WorkflowWNodeMenuEditComponent {

    // Project that contains the workflow
    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() popup: IPopup;
    @Input() node: WNode;

    displayInputName = false;
    permissionEnum = PermissionValue;
    loading = false;

    // Modal
    @ViewChild('workflowDeleteNode')
    workflowDeleteNode: WorkflowDeleteNodeComponent;
    @ViewChild('workflowTrigger')
    workflowTrigger: WorkflowTriggerComponent;
    @ViewChild('workflowEditOutgoingHook')
    workflowEditOutgoingHook: WorkflowNodeOutGoingHookEditComponent;
    @ViewChild('workflowAddHook')
    workflowAddHook: WorkflowHookModalComponent;
    @ViewChild('nodeEditModal')
    nodeEditModal: WorkflowNodeEditModalComponent;


    // Subscription
    nameWarning: WorkflowPipelineNameImpact;

    constructor(
        private store: Store,
        private _workflowCoreService: WorkflowCoreService,
        private _toast: ToastService,
        private _translate: TranslateService,
    ) {

    }

    canEdit(): boolean {
        return this.workflow.permission === PermissionValue.READ_WRITE_EXECUTE;
    }

    rename(): void {
        if (!this.canEdit()) {
            return;
        }
        let clonedWorkflow: Workflow = cloneDeep(this.workflow);
        let node = Workflow.getNodeByID(this.node.id, clonedWorkflow);
        if (!node) {
            return;
        }
        node.name = this.node.name;
        node.ref = this.node.name;
        // Update join
        if (clonedWorkflow.workflow_data.joins) {
            clonedWorkflow.workflow_data.joins.forEach(j => {
                for (let i = 0; i < j.parents.length; i++) {
                    if (j.parents[i].parent_id === node.id) {
                        j.parents[i].parent_name = node.name;
                        break;
                    }
                }
            });
        }

        this.updateWorkflow(clonedWorkflow, null);
    }

    openRenameArea(): void {
        if (!this.canEdit()) {
            return;
        }
        this.nameWarning = Workflow.getNodeNameImpact(this.workflow, this.node.name);
        this.displayInputName = true;
    }

    openDeleteNodeModal(): void {
        this.popup.close();
        if (!this.canEdit()) {
            return;
        }
        if (this.workflowDeleteNode) {
            this.workflowDeleteNode.show();
        }
    }

    openEditNodeModal(): void {
        this.popup.close();
        if (!this.canEdit()) {
            return;
        }
        if (this.nodeEditModal) {
            this.nodeEditModal.show();
        }
    }

    openTriggerModal(t: string, parent: boolean): void {
        this.popup.close();
        if (!this.canEdit()) {
            return;
        }
        if (this.workflowTrigger) {
            this.workflowTrigger.show(t, parent);
        }
    }

    openEditOutgoingHookModal(): void {
        this.popup.close();
        if (!this.canEdit()) {
            return;
        }
        if (this.workflowEditOutgoingHook) {
            this.workflowEditOutgoingHook.show();
        }
    }

    openAddHookModal(): void {
        this.popup.close();
        if (this.canEdit() && this.workflowAddHook) {
            this.workflowAddHook.show();
        }
    }

    createFork(): void {
        let n = Workflow.getNodeByID(this.node.id, this.workflow);
        let fork = new WNode();
        fork.type = WNodeType.FORK;
        let t = new WNodeTrigger();
        t.child_node = fork;
        t.parent_node_id = n.id;
        t.parent_node_name = n.ref;

        this.loading = true;
        this.store.dispatch(new AddNodeTriggerWorkflow({
            projectKey: this.project.key,
            workflowName: this.workflow.name,
            parentId: this.node.id,
            trigger: t
        })).pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this.popup.close();
            });
    }

    createJoin(): void {
        let join = new WNode();
        join.type = WNodeType.JOIN;
        join.parents = new Array<WNodeJoin>();
        let p = new WNodeJoin();
        p.parent_id = this.node.id;
        p.parent_name = this.node.ref;
        join.parents.push(p);

        this.loading = true;
        this.store.dispatch(new AddJoinWorkflow({
            projectKey: this.project.key,
            workflowName: this.workflow.name,
            join
        })).pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this.popup.close();
            });
    }

    updateWorkflow(w: Workflow, modal: ActiveModal<boolean, boolean, void>): void {
        this.loading = true;
        this.store.dispatch(new UpdateWorkflow({
            projectKey: this.project.key,
            workflowName: this.workflow.name,
            changes: w
        })).pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('workflow_updated'));
                if (modal) {
                    modal.approve(null);
                }
            }, () => {
                if (Array.isArray(this.node.hooks) && this.node.hooks.length) {
                    this.node.hooks.pop();
                }
            });
    }

    updateHook(hook: WNodeHook, modal: ActiveModal<boolean, boolean, void>): void {
        this.loading = true;

        let action = new UpdateHookWorkflow({
            projectKey: this.project.key,
            workflowName: this.workflow.name,
            hook
        });

        if (!hook.uuid) {
            action = new AddHookWorkflow({
                projectKey: this.project.key,
                workflowName: this.workflow.name,
                hook
            });
        }

        this.store.dispatch(action).pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('workflow_updated'));
                if (modal) {
                    modal.approve(null);
                }
            });
    }

    linkJoin(): void {
        this.popup.close();
        if (!this.canEdit()) {
            return;
        }
        this._workflowCoreService.linkJoinEvent(this.node);
    }
}
