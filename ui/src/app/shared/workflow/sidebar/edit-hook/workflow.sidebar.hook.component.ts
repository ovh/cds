import {Component, Input, OnInit, ViewChild} from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import {cloneDeep} from 'lodash';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {finalize} from 'rxjs/operators';
import {Subscription} from 'rxjs/Subscription';
import {PermissionValue} from '../../../../model/permission.model';
import {Project} from '../../../../model/project.model';
import {HookStatus, TaskExecution, WorkflowHookTask} from '../../../../model/workflow.hook.model';
import {WNode, WNodeHook, Workflow} from '../../../../model/workflow.model';
import {HookService} from '../../../../service/hook/hook.service';
import {WorkflowEventStore} from '../../../../service/workflow/workflow.event.store';
import {WorkflowStore} from '../../../../service/workflow/workflow.store';
import {AutoUnsubscribe} from '../../../decorator/autoUnsubscribe';
import {DeleteModalComponent} from '../../../modal/delete/delete.component';
import {ToastService} from '../../../toast/ToastService';
import {WorkflowHookModalComponent} from '../../modal/hook-modal/hook.modal.component';
import {WorkflowNodeHookDetailsComponent} from '../../node/hook/details/hook.details.component';

@Component({
    selector: 'app-workflow-sidebar-hook',
    templateUrl: './workflow.sidebar.hook.component.html',
    styleUrls: ['./workflow.sidebar.hook.component.scss']
})
@AutoUnsubscribe()
export class WorkflowSidebarHookComponent implements OnInit {

    @Input() project: Project;
    @Input() workflow: Workflow;

    node: WNode;
    hook: WNodeHook;
    subHook: Subscription;

    loading = false;
    hookDetails: WorkflowHookTask;

    @ViewChild('deleteHookModal')
    deleteHookModal: DeleteModalComponent;
    @ViewChild('workflowDetailsHook')
    workflowDetailsHook: WorkflowNodeHookDetailsComponent;

    permissionEnum = PermissionValue;
    hookStatus = HookStatus;


    @ViewChild('workflowEditHook')
    workflowEditHook: WorkflowHookModalComponent;

    constructor(
        private _toast: ToastService, private _hookService: HookService, private _translate: TranslateService,
        private _workflowEventStore: WorkflowEventStore, private _workflowStore: WorkflowStore
    ) {
    }

    ngOnInit(): void {
        this.subHook = this._workflowEventStore.selectedHook().subscribe(h => {
            this.hook = h;
            if (this.hook) {
                this.node = Workflow.getNodeByID(this.hook.node_id, this.workflow);
                this.loadHookDetails();
            }
        });
    }

    loadHookDetails() {
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

    deleteHook() {
        let currentWorkflow = cloneDeep(this.workflow);
        for (let i = 0; i < currentWorkflow.workflow_data.node.hooks.length; i++) {
            if (currentWorkflow.workflow_data.node.hooks[i].uuid === this.hook.uuid) {
                currentWorkflow.workflow_data.node.hooks.splice(i, 1);
            }
        }
        this.updateWorkflow(currentWorkflow, this.deleteHookModal.modal);
    }

    updateWorkflow(w: Workflow, modal: ActiveModal<boolean, boolean, void>): void {
        this.loading = true;
        this._workflowStore.updateWorkflow(this.project.key, w)
            .pipe(finalize( () => {this.loading = false}))
            .subscribe(() => {
            this.loading = false;
            this._toast.success('', this._translate.instant('workflow_updated'));
            this._workflowEventStore.unselectAll();
            if (modal) {
                modal.approve(null);
            }
        });
    }

    openHookDetailsModal(taskExec: TaskExecution) {
        if (this.workflowDetailsHook && this.workflowDetailsHook.show) {
            this.workflowDetailsHook.show(taskExec);
        }
    }
}
