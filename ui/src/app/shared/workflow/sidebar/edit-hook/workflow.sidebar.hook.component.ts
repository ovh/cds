import { Component, Input, OnInit, ViewChild } from '@angular/core';
import { finalize } from 'rxjs/operators';
import { Subscription } from 'rxjs/Subscription';
import { PermissionValue } from '../../../../model/permission.model';
import { Project } from '../../../../model/project.model';
import { HookStatus, TaskExecution, WorkflowHookTask } from '../../../../model/workflow.hook.model';
import { WNode, WNodeHook, Workflow } from '../../../../model/workflow.model';
import { HookService } from '../../../../service/hook/hook.service';
import { WorkflowEventStore } from '../../../../service/workflow/workflow.event.store';
import { AutoUnsubscribe } from '../../../decorator/autoUnsubscribe';
import { DeleteModalComponent } from '../../../modal/delete/delete.component';
import { WorkflowNodeHookDetailsComponent } from '../../node/hook/details/hook.details.component';

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

    constructor(
        private _hookService: HookService,
        private _workflowEventStore: WorkflowEventStore,
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

    openHookDetailsModal(taskExec: TaskExecution) {
        if (this.workflowDetailsHook && this.workflowDetailsHook.show) {
            this.workflowDetailsHook.show(taskExec);
        }
    }
}
