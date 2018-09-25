import {Component, Input, OnInit, ViewChild} from '@angular/core';
import {PipelineStatus} from 'app/model/pipeline.model';
import {Project} from 'app/model/project.model';
import {HookStatus, TaskExecution, WorkflowHookTask} from 'app/model/workflow.hook.model';
import {Workflow, WorkflowNode, WorkflowNodeHook, WorkflowNodeOutgoingHook} from 'app/model/workflow.model';
import {WorkflowNodeOutgoingHookRun, WorkflowNodeRun, WorkflowRun} from 'app/model/workflow.run.model';
import {finalize} from 'rxjs/operators';
import {HookService} from '../../../../../service/hook/hook.service';
import {WorkflowEventStore} from '../../../../../service/workflow/workflow.event.store';
import {AutoUnsubscribe} from '../../../../../shared/decorator/autoUnsubscribe';
import {WorkflowNodeHookDetailsComponent} from '../../../../../shared/workflow/node/hook/details/hook.details.component';
import {WorkflowNodeHookFormComponent} from '../../../../../shared/workflow/node/hook/form/hook.form.component';

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
    @ViewChild('workflowConfigOutgoingHook')
    workflowConfigOutgoingHook: WorkflowNodeHookFormComponent;

    @ViewChild('workflowDetailsHook')
    workflowDetailsHook: WorkflowNodeHookDetailsComponent;

    loading = false;
    node: WorkflowNode;
    hookStatus = HookStatus;
    hookDetails: WorkflowHookTask;
    hook: WorkflowNodeHook;
    wr: WorkflowRun;
    nodeRun: WorkflowNodeRun;
    outgoingHook: WorkflowNodeOutgoingHook;
    outgoingHookRuns: Array<WorkflowNodeOutgoingHookRun>;
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
            } else if (this.wr && this.outgoingHook) {
                this.loadOutgoingHookDetails();
            }
            if (this.wr && this.wr.nodes && this.node && this.wr.nodes[this.node.id] && this.wr.nodes[this.node.id].length > 0) {
                this.nodeRun = this.wr.nodes[this.node.id][0];
            } else {
                this.nodeRun = null;
            }
        });
        this._workflowEventStore.selectedOutgoingHook().subscribe(oh => {
            this.outgoingHook = oh;
            if (this.wr && this.outgoingHook) {
                this.loadOutgoingHookDetails();
            }
        });
        this._workflowEventStore.outgoingHookEvents().subscribe(
            hr => {
                if (hr && this.outgoingHook && this.outgoingHook.id === hr.workflow_node_outgoing_hook_id) {
                    if (this.wr && this.wr.id === hr.workflow_run_id && this.wr.num === hr.num) {
                        if (!this.outgoingHookRuns) {
                            this.outgoingHookRuns = new Array<WorkflowNodeOutgoingHookRun>();
                        }
                        let found = false;
                        for (let i = 0; i < this.outgoingHookRuns.length; i++) {
                            if (this.outgoingHookRuns[i].subnumber === hr.subnumber) {
                                this.outgoingHookRuns[i] = hr;
                                found = true;
                            }
                        }
                        if (!found) {
                            this.outgoingHookRuns.unshift(hr);
                        }
                    }
                }
            }
        );
    }

    loadOutgoingHookDetails() {
        if (this.wr.outgoing_hooks &&  this.outgoingHook) {
            if (this.wr.outgoing_hooks[this.outgoingHook.id]) {
                this.outgoingHookRuns = this.wr.outgoing_hooks[this.outgoingHook.id];
                this.node = Workflow.findNode(this.wr.workflow, (node) => {
                    return node.outgoing_hooks.find(h => h.id === this.outgoingHook.id );
                });
            }
        }
    }

    loadHookDetails() {
        let hookId = this.hook.id;
        // Find node linked to this hook
        this.node = Workflow.findNode(this.wr.workflow, (node) => {
            return Array.isArray(node.hooks) && node.hooks.length &&
                node.hooks.find((h) => h.id === hookId);
        });

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
        if (this.workflowConfigHook && this.workflowConfigHook.show) {
            this.workflowConfigHook.show();
        }
    }
    openHookDetailsModal(taskExec: TaskExecution) {
        if (this.workflowDetailsHook && this.workflowDetailsHook.show) {
            this.workflowDetailsHook.show(taskExec);
        }
    }

    openOutgoingHookDetailsModal(hr: WorkflowNodeOutgoingHookRun) {
        if (this.workflowDetailsHook && this.workflowDetailsHook.show) {
            this.workflowDetailsHook.showOutgoingHook(hr);
        }
    }
}
