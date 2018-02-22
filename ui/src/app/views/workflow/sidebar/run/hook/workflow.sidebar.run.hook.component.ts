import {Component, Input, ViewChild} from '@angular/core';
import {Workflow, WorkflowNode, WorkflowNodeHook} from '../../../../../model/workflow.model';
import {WorkflowHookTask, HookStatus, TaskExecution} from '../../../../../model/workflow.hook.model';
import {AutoUnsubscribe} from '../../../../../shared/decorator/autoUnsubscribe';
import {WorkflowNodeHookFormComponent} from '../../../../../shared/workflow/node/hook/form/hook.form.component';
import {WorkflowNodeHookDetailsComponent} from '../../../../../shared/workflow/node/hook/details/hook.details.component';
import {Project} from '../../../../../model/project.model';
import {HookService} from '../../../../../service/hook/hook.service';
import {finalize} from 'rxjs/operators';

@Component({
    selector: 'app-workflow-sidebar-run-hook',
    templateUrl: './workflow.sidebar.run.hook.component.html',
    styleUrls: ['./workflow.sidebar.run.hook.component.scss']
})
@AutoUnsubscribe()
export class WorkflowSidebarRunHookComponent {

    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() runNumber: number;
    @Input() readonly = false;
    @Input('hook')
    set hook(data: WorkflowNodeHook) {
        this._hook = data;
        if (data) {
            this.loadHookDetails();
        }
    }
    get hook() {
      return this._hook;
    }

    @ViewChild('workflowConfigHook')
    workflowConfigHook: WorkflowNodeHookFormComponent;
    @ViewChild('workflowDetailsHook')
    workflowDetailsHook: WorkflowNodeHookDetailsComponent;

    loading = false;
    node: WorkflowNode;
    hookStatus = HookStatus;
    hookDetails: WorkflowHookTask;
    _hook: WorkflowNodeHook;

    constructor(private _hookService: HookService) {

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
                    let found = false;
                    hook.executions = hook.executions.map((exec) => {
                        if (exec.nb_errors > 0) {
                            exec.status = HookStatus.FAIL;
                        }
                        if (!found && exec.workflow_run === this.runNumber) {
                            found = true;
                        }
                        return exec;
                    });

                    if (found) {
                        hook.executions = hook.executions.filter((h) => h.workflow_run === this.runNumber);
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
}
