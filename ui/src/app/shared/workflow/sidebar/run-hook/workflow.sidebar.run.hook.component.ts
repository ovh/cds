import { Component, Input, OnInit, ViewChild } from '@angular/core';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { HookStatus, TaskExecution, WorkflowHookTask } from 'app/model/workflow.hook.model';
import { WorkflowNodeRun, WorkflowNodeRunHookEvent, WorkflowRun } from 'app/model/workflow.run.model';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';
import { WNodeHook } from '../../../../model/workflow.model';
import { HookService } from '../../../../service/hook/hook.service';
import { WorkflowEventStore } from '../../../../service/workflow/workflow.event.store';
import { AutoUnsubscribe } from '../../../decorator/autoUnsubscribe';
import { WorkflowNodeHookDetailsComponent } from '../../node/hook/details/hook.details.component';
import { WorkflowNodeHookFormComponent } from '../../node/hook/form/hook.form.component';

@Component({
    selector: 'app-workflow-sidebar-run-hook',
    templateUrl: './workflow.sidebar.run.hook.component.html',
    styleUrls: ['./workflow.sidebar.run.hook.component.scss']
})
@AutoUnsubscribe()
export class WorkflowSidebarRunHookComponent implements OnInit {
    @ViewChild('workflowConfigHook')
    workflowConfigHook: WorkflowNodeHookFormComponent;

    @ViewChild('workflowDetailsHook')
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
    _hookSelectedSub: Subscription;
    _runSelectedSub: Subscription;

    constructor(
        private _hookService: HookService,
        private _workflowEventStore: WorkflowEventStore
    ) { }

    ngOnInit(): void {
        this._hookSelectedSub = this._workflowEventStore.selectedHook().subscribe(h => {
            this.hook = h;
            this.loadHookDetails();
        });
        this._runSelectedSub = this._workflowEventStore.selectedRun().subscribe(r => {
            this.wr = r;
            this.loadHookDetails();
        });
    }

    loadHookDetails() {
        if (this.wr && this.hook) {
            this.loading = true;
            this._hookService.getHookLogs(this.project.key, this.wr.workflow.name, this.hook.uuid)
                .pipe(finalize(() => this.loading = false))
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
