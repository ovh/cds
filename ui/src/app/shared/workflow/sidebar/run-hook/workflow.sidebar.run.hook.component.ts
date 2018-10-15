import {Component, Input, OnInit, ViewChild} from '@angular/core';
import {PipelineStatus} from 'app/model/pipeline.model';
import {Project} from 'app/model/project.model';
import {HookStatus, TaskExecution, WorkflowHookTask} from 'app/model/workflow.hook.model';
import {WNode} from 'app/model/workflow.model';
import {WorkflowNodeRun, WorkflowRun} from 'app/model/workflow.run.model';
import {finalize} from 'rxjs/operators';
import {WNodeHook} from '../../../../model/workflow.model';
import {HookService} from '../../../../service/hook/hook.service';
import {WorkflowEventStore} from '../../../../service/workflow/workflow.event.store';
import {AutoUnsubscribe} from '../../../decorator/autoUnsubscribe';
import {WorkflowNodeHookDetailsComponent} from '../../node/hook/details/hook.details.component';
import {WorkflowNodeHookFormComponent} from '../../node/hook/form/hook.form.component';

@Component({
    selector: 'app-workflow-sidebar-run-hook',
    templateUrl: './workflow.sidebar.run.hook.component.html',
    styleUrls: ['./workflow.sidebar.run.hook.component.scss']
})
@AutoUnsubscribe()
export class WorkflowSidebarRunHookComponent implements OnInit {

    @Input() project: Project;

    @ViewChild('workflowConfigHook')
    workflowConfigHook: WorkflowNodeHookFormComponent;

    @ViewChild('workflowDetailsHook')
    workflowDetailsHook: WorkflowNodeHookDetailsComponent;

    loading = false;
    node: WNode;
    hookStatus = HookStatus;
    hookDetails: WorkflowHookTask;
    hook: WNodeHook;
    wr: WorkflowRun;
    nodeRun: WorkflowNodeRun;
    pipelineStatusEnum = PipelineStatus;

    constructor(private _hookService: HookService, private _workflowEventStore: WorkflowEventStore) {
    }

    ngOnInit(): void {
        this._workflowEventStore.selectedHook().subscribe(h => {
            this.hook = h;
            if (this.hook && this.wr) {
                this.loadHookDetails();
            }
        });
        this._workflowEventStore.selectedRun().subscribe(r => {
            this.wr = r;
            if (this.wr && this.hook) {
                this.loadHookDetails();
            }
            if (this.wr && this.wr.nodes && this.node && this.wr.nodes[this.node.id] && this.wr.nodes[this.node.id].length > 0) {
                this.nodeRun = this.wr.nodes[this.node.id][0];
            } else {
                this.nodeRun = null;
            }
        });
    }

    loadHookDetails() {
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
    }

    openHookConfigModal() {
        /*
        if (this.workflowConfigHook && this.workflowConfigHook.show) {
            this.workflowConfigHook.show();
        }
        */
    }
    openHookDetailsModal(taskExec: TaskExecution) {
        if (this.workflowDetailsHook && this.workflowDetailsHook.show) {
            this.workflowDetailsHook.show(taskExec);
        }
    }
}
