import { Component, Input, OnInit, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { FetchPipeline } from 'app/store/pipelines.action';
import { PipelinesState } from 'app/store/pipelines.state';
import { AddHookWorkflow, AddJoinWorkflow, AddNodeTriggerWorkflow, UpdateHookWorkflow, UpdateWorkflow } from 'app/store/workflows.action';
import { cloneDeep } from 'lodash';
import {IPopup, ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import { ActiveModal } from 'ng2-semantic-ui/dist';
import { finalize, flatMap } from 'rxjs/operators';
import { PermissionValue } from '../../../../model/permission.model';
import { Project } from '../../../../model/project.model';
import * as workflowModel from '../../../../model/workflow.model';
import { WorkflowCoreService } from '../../../../service/services.module';
import { AutoUnsubscribe } from '../../../decorator/autoUnsubscribe';
import { ToastService } from '../../../toast/ToastService';
import { WorkflowNodeConditionsComponent } from '../../modal/conditions/node.conditions.component';
import { WorkflowNodeContextComponent } from '../../modal/context/workflow.node.context.component';
import { WorkflowDeleteNodeComponent } from '../../modal/delete/workflow.node.delete.component';
import { WorkflowHookModalComponent } from '../../modal/hook-modal/hook.modal.component';
import { WorkflowNodeOutGoingHookEditComponent } from '../../modal/outgoinghook-edit/outgoinghook.edit.component';
import { WorkflowNodePermissionsComponent } from '../../modal/permissions/node.permissions.component';
import { WorkflowTriggerComponent } from '../../modal/trigger/workflow.trigger.component';
import {WNode} from "../../../../model/workflow.model";
import {WorkflowNodeTriggerComponent} from '@cds/shared/workflow/modal/node-trigger/node.trigger.component';

@Component({
    selector: 'app-workflow-menu-wnode-edit',
    templateUrl: './menu.edit.node.html',
    styleUrls: ['./menu.edit.node.scss']
})
@AutoUnsubscribe()
export class WorkflowWNodeMenuEditComponent {

    // Project that contains the workflow
    @Input() project: Project;
    @Input() workflow: workflowModel.Workflow;
    @Input() popup: IPopup;
    @Input() node: WNode;

    displayInputName = false;
    permissionEnum = PermissionValue;
    loading = false;

    // Modal
    @ViewChild('workflowDeleteNode')
    workflowDeleteNode: WorkflowDeleteNodeComponent;
    @ViewChild('workflowContext')
    workflowContext: WorkflowNodeContextComponent;
    @ViewChild('workflowConditions')
    workflowConditions: WorkflowNodeConditionsComponent;
    @ViewChild('workflowTrigger')
    workflowTrigger: WorkflowTriggerComponent;
    @ViewChild('workflowEditOutgoingHook')
    workflowEditOutgoingHook: WorkflowNodeOutGoingHookEditComponent;
    @ViewChild('workflowAddHook')
    workflowAddHook: WorkflowHookModalComponent;
    @ViewChild('workflowNodePermissions')
    workflowNodePermissions: WorkflowNodePermissionsComponent;
    @ViewChild('nodeNameWarningModal')
    nodeNameWarningModal: ModalTemplate<boolean, boolean, void>;

    // Subscription
    nameWarning: workflowModel.WorkflowPipelineNameImpact;

    constructor(
        private store: Store,
        private _workflowCoreService: WorkflowCoreService,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _modalService: SuiModalService
    ) {

    }

    canEdit(): boolean {
        return this.workflow.permission === PermissionValue.READ_WRITE_EXECUTE;
    }

    rename(): void {
        if (!this.canEdit()) {
            return;
        }
        let clonedWorkflow: workflowModel.Workflow = cloneDeep(this.workflow);
        let node = workflowModel.Workflow.getNodeByID(this.node.id, clonedWorkflow);
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
        this.nameWarning = workflowModel.Workflow.getNodeNameImpact(this.workflow, this.node.name);
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

    openWarningModal(): void {
        let tmpl = new TemplateModalConfig<boolean, boolean, void>(this.nodeNameWarningModal);
        this._modalService.open(tmpl);
    }

    openEditContextModal(): void {
        this.store.dispatch(new FetchPipeline({
            projectKey: this.project.key,
            pipelineName: this.workflow.pipelines[this.node.context.pipeline_id].name
        })).pipe(
            flatMap(() => this.store.selectOnce(PipelinesState.selectPipeline(
                this.project.key, this.workflow.pipelines[this.node.context.pipeline_id].name
            )))
        ).subscribe((pip) => {
            if (pip) {
                setTimeout(() => {
                    this.workflowContext.show();
                }, 100);
            }
        });
    }

    openEditRunConditions(): void {
        this.workflowConditions.show();
    }

    openNodePermissions(): void {
        this.workflowNodePermissions.show();
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
        if (this.canEdit() && this.workflowAddHook) {
            this.workflowAddHook.show();
        }
    }

    createFork(): void {
        let n = workflowModel.Workflow.getNodeByID(this.node.id, this.workflow);
        let fork = new workflowModel.WNode();
        fork.type = workflowModel.WNodeType.FORK;
        let t = new workflowModel.WNodeTrigger();
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
        let join = new workflowModel.WNode();
        join.type = workflowModel.WNodeType.JOIN;
        join.parents = new Array<workflowModel.WNodeJoin>();
        let p = new workflowModel.WNodeJoin();
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

    updateWorkflow(w: workflowModel.Workflow, modal: ActiveModal<boolean, boolean, void>): void {
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

    updateHook(hook: workflowModel.WNodeHook, modal: ActiveModal<boolean, boolean, void>): void {
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
        if (!this.canEdit()) {
            return;
        }
        this._workflowCoreService.linkJoinEvent(this.node);
    }
}
