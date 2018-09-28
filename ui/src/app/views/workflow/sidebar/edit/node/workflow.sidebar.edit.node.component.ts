import {Component, Input, OnInit, ViewChild} from '@angular/core';
import {Router} from '@angular/router';
import {TranslateService} from '@ngx-translate/core';
import {cloneDeep} from 'lodash';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {Subscription} from 'rxjs';
import {PermissionValue} from '../../../../../model/permission.model';
import {Project} from '../../../../../model/project.model';
import {
    WNode, WNodeHook,
    Workflow,
    WorkflowNode,
    WorkflowNodeTrigger,
    WorkflowPipelineNameImpact,
} from '../../../../../model/workflow.model';
import {WorkflowEventStore} from '../../../../../service/workflow/workflow.event.store';
import {WorkflowStore} from '../../../../../service/workflow/workflow.store';
import {AutoUnsubscribe} from '../../../../../shared/decorator/autoUnsubscribe';
import {ToastService} from '../../../../../shared/toast/ToastService';
import {WorkflowTriggerComponent} from '../../../../../shared/workflow/modal/trigger/workflow.trigger.component';
import {WorkflowNodeHookFormComponent} from '../../../../../shared/workflow/node/hook/form/hook.form.component';
import {HookEvent} from '../../../../../shared/workflow/node/hook/hook.event';

@Component({
    selector: 'app-workflow-sidebar-edit-node',
    templateUrl: './workflow.sidebar.edit.node.component.html',
    styleUrls: ['./workflow.sidebar.edit.node.component.scss']
})
@AutoUnsubscribe()
export class WorkflowSidebarEditNodeComponent implements OnInit {

    // Project that contains the workflow
    @Input() project: Project;
    @Input() workflow: Workflow;

    nodeSub: Subscription;

    // Child component
    @ViewChild('workflowTriggerParent')
    workflowTriggerParent: WorkflowTriggerComponent;
    @ViewChild('worklflowAddHook')
    worklflowAddHook: WorkflowNodeHookFormComponent;
    @ViewChild('nodeNameWarningModal')
    nodeNameWarningModal: ModalTemplate<boolean, boolean, void>;

    // Modal
    @ViewChild('nodeParentModal')
    nodeParentModal: ModalTemplate<boolean, boolean, void>;
    newParentNode: WorkflowNode;
    newTrigger: WorkflowNode = new WorkflowNode();
    node: WNode;
    previousNodeName: string;
    displayInputName = false;
    loading = false;
    nameWarning: WorkflowPipelineNameImpact;
    permissionEnum = PermissionValue;
    isChildOfOutgoingHook = false;

    constructor(private _workflowStore: WorkflowStore, private _translate: TranslateService, private _toast: ToastService,
                private _modalService: SuiModalService,
                private _router: Router, private _workflowEventStore: WorkflowEventStore) {}

    ngOnInit(): void {
        this.nodeSub = this._workflowEventStore.selectedNode().subscribe(n => {
            if (n) {
                if (!this.displayInputName) {
                    this.previousNodeName = n.name
                }
                if (this.workflow) {
                    this.isChildOfOutgoingHook = Workflow.isChildOfOutgoingHook(this.workflow, null, null, n.id);
                }
            }
            this.node = n;
        });
    }

    canEdit(): boolean {
        return this.workflow.permission === PermissionValue.READ_WRITE_EXECUTE;
    }

    openAddHookModal(): void {
        if (this.canEdit() && this.worklflowAddHook) {
            this.worklflowAddHook.show();
        }
    }

    openAddParentModal(): void {
        if (!this.canEdit()) {
            return;
        }
        this.newParentNode = new WorkflowNode();
        if (this.workflowTriggerParent) {
          this.workflowTriggerParent.show('');
        }
    }

    addNewParentNode(): void {
        let workflowToUpdate = cloneDeep(this.workflow);
        let oldRoot = cloneDeep(this.workflow.root);
        workflowToUpdate.root = this.newParentNode;
        if (oldRoot.hooks) {
            workflowToUpdate.root.hooks = oldRoot.hooks;
        }
        delete oldRoot.hooks;
        workflowToUpdate.root.triggers = new Array<WorkflowNodeTrigger>();
        let t = new WorkflowNodeTrigger();
        t.workflow_dest_node = oldRoot;
        workflowToUpdate.root.triggers.push(t);

        this.updateWorkflow(workflowToUpdate, this.workflowTriggerParent.modal);
    }

    openWarningModal(): void {
        let tmpl = new TemplateModalConfig<boolean, boolean, void>(this.nodeNameWarningModal);
        this._modalService.open(tmpl);
    }

    deleteWorkflow(w: Workflow, modal: ActiveModal<boolean, boolean, void>): void {
        this._workflowStore.deleteWorkflow(this.project.key, w).subscribe(() => {
            this._toast.success('', this._translate.instant('workflow_deleted'));
            modal.approve(true);
            this._workflowEventStore.unselectAll();
            this._router.navigate(['/project', this.project.key], { queryParams: { tab: 'workflows'}});
        });
    }

    updateWorkflow(w: Workflow, modal?: ActiveModal<boolean, boolean, void>): void {
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


    openRenameArea(): void {
        if (!this.canEdit()) {
            return;
        }
        this.nameWarning = Workflow.getNodeNameImpact(this.workflow, this.node.name);
        this.displayInputName = true;
    }



    addHook(he: HookEvent): void {
        if (!this.canEdit()) {
            return;
        }
        if (!this.node.hooks) {
            this.node.hooks = new Array<WNodeHook>();
        }
        this.node.hooks.push(he.hook);
        this.updateWorkflow(this.workflow, null);
    }

    addOutgoingHook(he: HookEvent): void {
        /*
        if (!this.canEdit()) {
            return;
        }
        if (!this.node.triggers) {
            this.node.outgoing_hooks = new Array<WorkflowNodeOutgoingHook>();
        }
        let oh = new WorkflowNodeOutgoingHook();
        oh.config = he.hook.config
        oh.id = he.hook.id
        oh.model = he.hook.model
        oh.name = he.name;
        this.node.outgoing_hooks.push(oh);
        this.updateWorkflow(this.workflow, this.worklflowAddOutgoingHook.modal);
         */
        // TODO
    }
}
