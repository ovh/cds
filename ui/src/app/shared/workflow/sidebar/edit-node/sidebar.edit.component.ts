import { Component, Input, OnInit, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { cloneDeep } from 'lodash';
import { ModalTemplate, SuiModalService, TemplateModalConfig } from 'ng2-semantic-ui';
import { ActiveModal } from 'ng2-semantic-ui/dist';
import { Subscription } from 'rxjs';
import { first } from 'rxjs/operators';
import { PermissionValue } from '../../../../model/permission.model';
import { Project } from '../../../../model/project.model';
import { WNode, WNodeJoin, WNodeTrigger, WNodeType, Workflow, WorkflowPipelineNameImpact } from '../../../../model/workflow.model';
import { PipelineStore, WorkflowCoreService, WorkflowEventStore, WorkflowStore } from '../../../../service/services.module';
import { AutoUnsubscribe } from '../../../decorator/autoUnsubscribe';
import { ToastService } from '../../../toast/ToastService';
import { WorkflowNodeConditionsComponent } from '../../modal/conditions/node.conditions.component';
import { WorkflowNodeContextComponent } from '../../modal/context/workflow.node.context.component';
import { WorkflowDeleteNodeComponent } from '../../modal/delete/workflow.node.delete.component';
import { WorkflowHookModalComponent } from '../../modal/hook-modal/hook.modal.component';
import { WorkflowNodeOutGoingHookEditComponent } from '../../modal/outgoinghook-edit/outgoinghook.edit.component';
import { WorkflowNodePermissionsComponent } from '../../modal/permissions/node.permissions.component';
import { WorkflowTriggerComponent } from '../../modal/trigger/workflow.trigger.component';

@Component({
    selector: 'app-workflow-sidebar-wnode-edit',
    templateUrl: './sidebar.edit.html',
    styleUrls: ['./sidebar.edit.scss']
})
@AutoUnsubscribe()
export class WorkflowWNodeSidebarEditComponent implements OnInit {

    // Project that contains the workflow
    @Input() project: Project;
    @Input() workflow: Workflow;

    node: WNode;
    nodeSub: Subscription;

    displayInputName = false;
    previousNodeName: string;

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
    pipelineSubscription: Subscription;
    nameWarning: WorkflowPipelineNameImpact;

    constructor(private _workflowEventStore: WorkflowEventStore, private _pipelineStore: PipelineStore,
                private _workflowCoreService: WorkflowCoreService, private _workflowStore: WorkflowStore, private _toast: ToastService,
                private _translate: TranslateService, private _modalService: SuiModalService) {

    }

    ngOnInit(): void {
        this.nodeSub = this._workflowEventStore.selectedNode().subscribe(n => {
            if (n) {
                if (!this.displayInputName) {
                    this.previousNodeName = n.name
                }
            }
            this.node = n;
        });
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
            clonedWorkflow.workflow_data.joins.forEach( j => {
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
       this.pipelineSubscription =
            this._pipelineStore.getPipelines(this.project.key,
                this.workflow.pipelines[this.node.context.pipeline_id].name)
                .pipe(first())
                .subscribe(pips => {
                if (pips.get(this.project.key + '-' + this.workflow.pipelines[this.node.context.pipeline_id].name)) {
                    setTimeout(() => {
                        this.workflowContext.show();
                        this.pipelineSubscription.unsubscribe();
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
        if (!this.canEdit()) {
            return;
        }
        if (this.workflowTrigger) {
            this.workflowTrigger.show(t, parent);
        }
    }

    openEditOutgoingHookModal(): void {
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
        let clonedWorkflow = cloneDeep(this.workflow);
        let n = Workflow.getNodeByID(this.node.id, clonedWorkflow);
        if (!n.triggers) {
            n.triggers = new Array<WNodeTrigger>();
        }
        let fork = new WNode();
        fork.type = WNodeType.FORK;
        let t = new WNodeTrigger();
        t.child_node = fork;
        t.parent_node_id = n.id;
        t.parent_node_name = n.ref;
        n.triggers.push(t);
        this.updateWorkflow(clonedWorkflow, null);
    }

    createJoin(): void {
        let clonedWorkflow = cloneDeep(this.workflow);
        let join = new WNode();
        join.type = WNodeType.JOIN;
        join.parents = new Array<WNodeJoin>();
        let p = new WNodeJoin();
        p.parent_id = this.node.id;
        p.parent_name = this.node.ref;
        join.parents.push(p);

        if (!clonedWorkflow.workflow_data.joins) {
            clonedWorkflow.workflow_data.joins = new Array<WNode>();
        }
        clonedWorkflow.workflow_data.joins.push(join);
        this.updateWorkflow(clonedWorkflow, null);
    }

    updateWorkflow(w: Workflow, modal: ActiveModal<boolean, boolean, void>): void {
        this.loading = true;
        this._workflowStore.updateWorkflow(this.project.key, w).subscribe(() => {
            this.loading = false;
            this._toast.success('', this._translate.instant('workflow_updated'));
            this._workflowEventStore.unselectAll();
            if (modal) {
                modal.approve(null);
            }
        }, () => {
            if (Array.isArray(this.node.hooks) && this.node.hooks.length) {
                this.node.hooks.pop();
            }
            this.loading = false;
        });
    }

    linkJoin(): void {
        if (!this.canEdit()) {
            return;
        }
        this._workflowCoreService.linkJoinEvent(this.node);
    }
}
