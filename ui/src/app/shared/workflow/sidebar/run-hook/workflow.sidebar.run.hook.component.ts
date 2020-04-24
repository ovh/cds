import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit, ViewChild } from '@angular/core';
import { Select } from '@ngxs/store';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { HookStatus, TaskExecution, WorkflowHookTask } from 'app/model/workflow.hook.model';
import { WNodeHook } from 'app/model/workflow.model';
import { WorkflowNodeRun, WorkflowNodeRunHookEvent, WorkflowRun } from 'app/model/workflow.run.model';
import { HookService } from 'app/service/hook/hook.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { WorkflowNodeHookDetailsComponent } from 'app/shared/workflow/node/hook/details/hook.details.component';
import { WorkflowState } from 'app/store/workflow.state';
import { Observable, Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-workflow-sidebar-run-hook',
    templateUrl: './workflow.sidebar.run.hook.component.html',
    styleUrls: ['./workflow.sidebar.run.hook.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowSidebarRunHookComponent implements OnInit {

    @ViewChild('workflowDetailsHook')
    workflowDetailsHook: WorkflowNodeHookDetailsComponent;

    @Input() project: Project;
    @Input() wname: string;

    loading = false;
    hookStatus = HookStatus;
    hookDetails: WorkflowHookTask;
    hook: WNodeHook;
    hookEvent: WorkflowNodeRunHookEvent;
    wr: WorkflowRun;
    nodeRun: WorkflowNodeRun;
    pipelineStatusEnum = PipelineStatus;
    subStore: Subscription;

    @Select(WorkflowState.getSelectedHook()) hook$: Observable<WNodeHook>;
    hookSubs: Subscription;
    @Select(WorkflowState.getSelectedWorkflowRun()) workflowRun$: Observable<WorkflowRun>;
    wrSubs: Subscription;

    constructor(
        private _hookService: HookService,
        private _cd: ChangeDetectorRef
    ) {}

    ngOnInit(): void {
        this.hookSubs = this.hook$.subscribe(h => {
            // if no hooks, return
            if (!h && !this.hook) {
                return;
            }
            // if same hooks return
            if (h && this.hook && h.uuid === this.hook.uuid) {
                return;
            }
            this.hookEvent = null;
            this.hook = h;
            this.loadHookDetails();
            this._cd.markForCheck();
        });
        this.wrSubs = this.workflowRun$.subscribe(workRun => {
            if (!workRun && !this.wr) {
                return;
            }
            if (workRun && this.wr && this.wr.id === workRun.id && this.hookEvent) {
                return
            }
            this.wr = workRun;
            this.loadHookDetails();
            this._cd.markForCheck();
        });
    }

    loadHookDetails() {
        if (this.wr && this.hook) {
            this.loading = true;
            this._hookService.getHookLogs(this.project.key, this.wname, this.hook.uuid)
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
