import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit, ViewChild } from '@angular/core';
import { Select } from '@ngxs/store';
import { Project } from 'app/model/project.model';
import { HookStatus, TaskExecution, WorkflowHookTask } from 'app/model/workflow.hook.model';
import { WNode, WNodeHook, Workflow } from 'app/model/workflow.model';
import { HookService } from 'app/service/hook/hook.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { WorkflowNodeHookDetailsComponent } from 'app/shared/workflow/node/hook/details/hook.details.component';
import { WorkflowState } from 'app/store/workflow.state';
import { Observable } from 'rxjs';
import { finalize } from 'rxjs/operators';
import { Subscription } from 'rxjs/Subscription';

@Component({
    selector: 'app-workflow-sidebar-hook',
    templateUrl: './workflow.sidebar.hook.component.html',
    styleUrls: ['./workflow.sidebar.hook.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowSidebarHookComponent implements OnInit {

    @Input() project: Project;
    @Input() workflow: Workflow;

    node: WNode;
    hook: WNodeHook;
    @Select(WorkflowState.getSelectedHook()) hooks$: Observable<WNodeHook>;
    subHook: Subscription;

    loading = false;
    hookDetails: WorkflowHookTask;

    @ViewChild('workflowDetailsHook', {static: false})
    workflowDetailsHook: WorkflowNodeHookDetailsComponent;

    hookStatus = HookStatus;

    constructor(
        private _hookService: HookService,
        private _cd: ChangeDetectorRef
    ) {
    }

    ngOnInit(): void {
        this.subHook = this.hooks$.subscribe((h: WNodeHook) => {
            this.hook = h;
            if (this.hook) {
                this.node = Workflow.getNodeByID(this.hook.node_id, this.workflow);
                this.loadHookDetails();
            }
            this._cd.markForCheck();
        });
    }

    loadHookDetails() {
        this.loading = true;
        this._hookService.getHookLogs(this.project.key, this.workflow.name, this.hook.uuid)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
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
