import {Component, Input, ViewChild} from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import { WorkflowTriggerComponent } from 'app/shared/workflow/modal/trigger/workflow.trigger.component';
import {cloneDeep} from 'lodash';
import {finalize} from 'rxjs/operators';
import {Subscription} from 'rxjs/Subscription';
import {PermissionValue} from '../../../../../model/permission.model';
import {Project} from '../../../../../model/project.model';
import {TaskExecution} from '../../../../../model/workflow.hook.model';
import {
    WNodeHook,
    Workflow,
    WorkflowNode,
    WorkflowNodeHook,
    WorkflowNodeOutgoingHook,
    WorkflowNodeTrigger
} from '../../../../../model/workflow.model';
import {WorkflowEventStore} from '../../../../../service/workflow/workflow.event.store';
import {WorkflowStore} from '../../../../../service/workflow/workflow.store';
import {AutoUnsubscribe} from '../../../../../shared/decorator/autoUnsubscribe';
import {DeleteModalComponent} from '../../../../../shared/modal/delete/delete.component';
import {ToastService} from '../../../../../shared/toast/ToastService';
import {WorkflowNodeHookDetailsComponent} from '../../../../../shared/workflow/node/hook/details/hook.details.component';
import {WorkflowNodeHookFormComponent} from '../../../../../shared/workflow/node/hook/form/hook.form.component';
import {HookEvent} from '../../../../../shared/workflow/node/hook/hook.event';

@Component({
    selector: 'app-workflow-sidebar-outgoing-hook',
    templateUrl: './workflow.sidebar.outgoinghook.component.html',
    styleUrls: ['./workflow.sidebar.outgoinghook.component.scss']
})
@AutoUnsubscribe()
export class WorkflowSidebarOutgoingHookComponent {

    @Input() project: Project;
    @Input() workflow: Workflow;

    hook: WorkflowNodeOutgoingHook;
    subHook: Subscription;

    @ViewChild('workflowEditHook')
    workflowEditHook: WorkflowNodeHookFormComponent;
    @ViewChild('deleteHookModal')
    deleteHookModal: DeleteModalComponent;
    @ViewChild('workflowDetailsHook')
    workflowDetailsHook: WorkflowNodeHookDetailsComponent;
    @ViewChild('workflowTrigger')
    workflowTrigger: WorkflowTriggerComponent;

    loading = false;
    node: WorkflowNode;
    _hook: WorkflowNodeHook;
    permissionEnum = PermissionValue;
    newTrigger: WorkflowNode;

    constructor(
        private _workflowStore: WorkflowStore,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _workflowEventStore: WorkflowEventStore
    ) {}

    openHookEditModal() {
        if (this.workflowEditHook && this.workflowEditHook.show) {
            this.workflowEditHook.show();
        }
    }

    openDeleteHookModal() {
        if (this.workflow.permission < PermissionValue.READ_WRITE_EXECUTE) {
          return;
        }
        if (this.deleteHookModal && this.deleteHookModal.show) {
            this.deleteHookModal.show();
        }
    }

    deleteOutgoingHook() {
        let hEvent = new HookEvent('delete', new WNodeHook());
        hEvent.hook.id = this.hook.id;
        this.updateOutgoingHook(hEvent);
    }

    updateOutgoingHook(h: HookEvent): void {
        let workflowToUpdate = cloneDeep(this.workflow);
        this.loading = true;
        if (h.type === 'delete') {
            Workflow.removeOutgoingHook(workflowToUpdate, h.hook.id);
        } else {
            Workflow.updateOutgoingHook(workflowToUpdate, h.hook.id, h.name, h.hook.config);
        }
        this._workflowStore.updateWorkflow(workflowToUpdate.project_key, workflowToUpdate)
            .pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                if (this.workflowEditHook && this.workflowEditHook.modal) {
                    this.workflowEditHook.modal.approve(true);
                }
                this._workflowEventStore.unselectAll();
                this._toast.success('', this._translate.instant('workflow_updated'));
            });
    }

    openHookDetailsModal(taskExec: TaskExecution) {
        if (this.workflowDetailsHook && this.workflowDetailsHook.show) {
            this.workflowDetailsHook.show(taskExec);
        }
    }

    openTriggerModal(): void {
        if (this.workflow.permission !== PermissionValue.READ_WRITE_EXECUTE) {
            return;
        }
        this.newTrigger = new WorkflowNode();
        if (this.workflowTrigger) {
            this.workflowTrigger.show();
        }
    }

    saveTrigger(): void {
        let workflowToUpdate = cloneDeep(this.workflow);
        this.loading = true;
        let h = Workflow.findOutgoingHook(workflowToUpdate, this.hook.id);
        if (h) {
            if (!h.triggers) {
                h.triggers = new Array<WorkflowNodeTrigger>();
            }
            let t = new WorkflowNodeTrigger();
            t.workflow_dest_node = this.newTrigger;
            h.triggers.push(t);

            this._workflowStore.updateWorkflow(workflowToUpdate.project_key, workflowToUpdate)
            .pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                if (this.workflowTrigger && this.workflowTrigger.modal) {
                    this.workflowTrigger.modal.approve(true);
                }
                this._workflowEventStore.unselectAll();
                this._toast.success('', this._translate.instant('workflow_updated'));
            });
        }
    }
}
