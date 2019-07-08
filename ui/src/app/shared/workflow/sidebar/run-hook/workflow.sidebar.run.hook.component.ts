import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit, ViewChild } from '@angular/core';
import {Store} from '@ngxs/store';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { HookStatus, TaskExecution, WorkflowHookTask } from 'app/model/workflow.hook.model';
import { WNodeHook } from 'app/model/workflow.model';
import { WorkflowNodeRun, WorkflowNodeRunHookEvent, WorkflowRun } from 'app/model/workflow.run.model';
import { HookService } from 'app/service/hook/hook.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { WorkflowNodeHookDetailsComponent } from 'app/shared/workflow/node/hook/details/hook.details.component';
import {WorkflowState, WorkflowStateModel} from 'app/store/workflow.state';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-workflow-sidebar-run-hook',
    templateUrl: './workflow.sidebar.run.hook.component.html',
    styleUrls: ['./workflow.sidebar.run.hook.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowSidebarRunHookComponent implements OnInit {

    @ViewChild('workflowDetailsHook', {static: false})
    workflowDetailsHook: WorkflowNodeHookDetailsComponent;

    @Input() project: Project;

    loading = false;
    hookStatus = HookStatus;
    hookDetails: WorkflowHookTask;
    hook: WNodeHook;
    hookEvent: WorkflowNodeRunHookEvent;
    wr: WorkflowRun;
    nodeRun: WorkflowNodeRun;
    pipelineStatusEnum = PipelineStatus;
    subStore: Subscription;

    constructor(
        private _hookService: HookService,
        private _store: Store,
        private _cd: ChangeDetectorRef
    ) { }

    ngOnInit(): void {
        this.subStore = this._store.select(WorkflowState.getCurrent()).subscribe((s: WorkflowStateModel) => {
            this.wr = s.workflowRun;
            this.hook = s.hook;
            this.loadHookDetails();
            this._cd.markForCheck();
        });
    }

    loadHookDetails() {
        if (this.wr && this.hook) {
            this.loading = true;
            this._hookService.getHookLogs(this.project.key, this.wr.workflow.name, this.hook.uuid)
                .pipe(finalize(() => {
                    this.loading = false;
                    this._cd.markForCheck();
                }))
                .subscribe((hook) => {
                    if (Array.isArray(hook.executions) && hook.executions.length) {
                        let found = false;
                        hook.executions = hook.executions.map((exec) => {
                            if (exec.nb_errors > 0) {
                                exec.status = HookStatus.FAIL;
                            }
                            if (!found && exec.workflow_run === this.wr.num) {
                                found = true;
                            }
                            return exec;
                        });

                        if (found) {
                            hook.executions = hook.executions.filter((h) => h.workflow_run === this.wr.num);
                        }
                    }
                    this.hookDetails = hook;
                });
            if (this.wr.nodes) {
                Object.keys(this.wr.nodes).forEach(k => {
                    let nr = this.wr.nodes[k][0];
                    if (nr.hook_event && nr.hook_event.uuid === this.hook.uuid) {
                        this.hookEvent = nr.hook_event;
                    }
                });
            }
        }
    }

    openHookDetailsModal(taskExec: TaskExecution) {
        if (this.workflowDetailsHook && this.workflowDetailsHook.show) {
            this.workflowDetailsHook.show(taskExec);
        }
    }
}
