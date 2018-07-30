import {Component, Input, OnInit, ViewChild} from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import {cloneDeep} from 'lodash';
import {finalize} from 'rxjs/operators';
import {Subscription} from 'rxjs/Subscription';
import {PermissionValue} from '../../../../../model/permission.model';
import {Project} from '../../../../../model/project.model';
import {HookStatus, TaskExecution, WorkflowHookTask} from '../../../../../model/workflow.hook.model';
import {Workflow, WorkflowNode, WorkflowNodeHook} from '../../../../../model/workflow.model';
import {HookService} from '../../../../../service/hook/hook.service';
import {WorkflowEventStore} from '../../../../../service/workflow/workflow.event.store';
import {WorkflowStore} from '../../../../../service/workflow/workflow.store';
import {AutoUnsubscribe} from '../../../../../shared/decorator/autoUnsubscribe';
import {DeleteModalComponent} from '../../../../../shared/modal/delete/delete.component';
import {ToastService} from '../../../../../shared/toast/ToastService';
import {WorkflowNodeHookDetailsComponent} from '../../../../../shared/workflow/node/hook/details/hook.details.component';
import {WorkflowNodeHookFormComponent} from '../../../../../shared/workflow/node/hook/form/hook.form.component';
import {HookEvent} from '../../../../../shared/workflow/node/hook/hook.event';

@Component({
    selector: 'app-workflow-sidebar-hook',
    templateUrl: './workflow.sidebar.hook.component.html',
    styleUrls: ['./workflow.sidebar.hook.component.scss']
})
@AutoUnsubscribe()
export class WorkflowSidebarHookComponent implements OnInit {

    @Input() project: Project;
    @Input() workflow: Workflow;

    hook: WorkflowNodeHook;
    subHook: Subscription;

    @ViewChild('workflowEditHook')
    workflowEditHook: WorkflowNodeHookFormComponent;
    @ViewChild('deleteHookModal')
    deleteHookModal: DeleteModalComponent;
    @ViewChild('workflowDetailsHook')
    workflowDetailsHook: WorkflowNodeHookDetailsComponent;

    loading = false;
    node: WorkflowNode;
    hookStatus = HookStatus;
    hookDetails: WorkflowHookTask;
    _hook: WorkflowNodeHook;
    permissionEnum = PermissionValue;

    constructor(
        private _workflowStore: WorkflowStore,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _hookService: HookService,
        private _workflowEventStore: WorkflowEventStore
    ) {
    }

    ngOnInit(): void {
        this.subHook = this._workflowEventStore.selectedHook().subscribe(h => {
            this.hook = h;
            if (this.hook) {
                this.loadHookDetails();
            }
        });
    }

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

    loadHookDetails() {
        let hookId = this.hook.id;
        // Find node linked to this hook
        this.node = Workflow.findNode(this.workflow, (node) => {
            return Array.isArray(node.hooks) && node.hooks.length &&
                node.hooks.find((h) => h.id === hookId);
        });

        this.loading = true;
        this._hookService.getHookLogs(this.project.key, this.workflow.name, this.hook.uuid)
            .pipe(finalize(() => this.loading = false))
            .subscribe((hook) => {
                if (Array.isArray(hook.executions) && hook.executions.length) {
                    hook.executions = hook.executions.map((exec) => {
                        if (exec.nb_errors > 0) {
                            exec.status = HookStatus.FAIL;
                        }
                        return exec;
                    });
                }
                this.hookDetails = hook;
            });
    }

    deleteHook() {
        let hEvent = new HookEvent('delete', this.hook);
        this.updateHook(hEvent);
    }

    updateHook(h: HookEvent): void {
        let workflowToUpdate = cloneDeep(this.workflow);
        this.loading = true;
        if (h.type === 'delete') {
            Workflow.removeHook(workflowToUpdate, h.hook);
        } else {
            Workflow.updateHook(workflowToUpdate, h.hook);
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
}
