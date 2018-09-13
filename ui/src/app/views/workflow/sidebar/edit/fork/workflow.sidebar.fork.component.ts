import {Component, Input, OnInit, ViewChild} from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import {cloneDeep} from 'lodash';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {Subscription} from 'rxjs/Subscription';
import {PermissionValue} from '../../../../../model/permission.model';
import {Project} from '../../../../../model/project.model';
import {
    Workflow,
    WorkflowNode,
    WorkflowNodeFork, WorkflowNodeForkTrigger
} from '../../../../../model/workflow.model';
import {WorkflowEventStore} from '../../../../../service/workflow/workflow.event.store';
import {WorkflowStore} from '../../../../../service/workflow/workflow.store';
import {AutoUnsubscribe} from '../../../../../shared/decorator/autoUnsubscribe';
import {ToastService} from '../../../../../shared/toast/ToastService';
import {WorkflowDeleteForkComponent} from '../../../../../shared/workflow/fork/delete/workflow.fork.delete.component';
import {WorkflowTriggerComponent} from '../../../../../shared/workflow/trigger/workflow.trigger.component';

@Component({
    selector: 'app-workflow-sidebar-fork',
    templateUrl: './workflow.sidebar.fork.component.html',
    styleUrls: ['./workflow.sidebar.fork.component.scss']
})
@AutoUnsubscribe()
export class WorkflowSidebarForkComponent implements OnInit {

    @Input() project: Project;
    @Input() workflow: Workflow;

    fork: WorkflowNodeFork;
    subFork: Subscription;
    newTrigger: WorkflowNode;

    previousNodeName: string;
    displayInputName = false;
    permissionEnum = PermissionValue;


    loading: boolean;

    @ViewChild('workflowTrigger')
    workflowTrigger: WorkflowTriggerComponent;
    @ViewChild('workflowDeleteFork')
    workflowDelete: WorkflowDeleteForkComponent;

    constructor(
        private _workflowStore: WorkflowStore,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _workflowEventStore: WorkflowEventStore
    ) {}

    ngOnInit(): void {
        this.subFork = this._workflowEventStore.selectedFork().subscribe(f => {
            this.fork = f;
            if (!this.displayInputName && f) {
                this.previousNodeName = f.name
            }
        });
    }

    openTriggerModal(): void {
        if (this.workflow.permission !== PermissionValue.READ_WRITE_EXECUTE) {
            return;
        }
        this.newTrigger = new WorkflowNode();
        if (this.workflowTrigger) {
            this.workflowTrigger.show();
        }
    }

    openRenameArea(): void {
        if (this.workflow.permission !== PermissionValue.READ_WRITE_EXECUTE) {
            return;
        }
        this.displayInputName = true;
    }

    saveTrigger(): void {
        if (this.workflow.permission !== PermissionValue.READ_WRITE_EXECUTE) {
            return null
        }
        let clonedWorkflow: Workflow = cloneDeep(this.workflow);
        let currentFork = Workflow.getForkByName(this.fork.name, clonedWorkflow);

        if (!currentFork) {
            return;
        }

        let t = new WorkflowNodeForkTrigger();
        t.workflow_node_fork_id = currentFork.id;
        t.workflow_dest_node = this.newTrigger;
        if (!currentFork.triggers) {
            currentFork.triggers = new Array<WorkflowNodeForkTrigger>();
        }
        currentFork.triggers.push(t);
        this.updateWorkflow(clonedWorkflow, this.workflowTrigger.modal, true);
    }

    rename(): void {
        if (this.workflow.permission !== PermissionValue.READ_WRITE_EXECUTE) {
            return;
        }
        let clonedWorkflow: Workflow = cloneDeep(this.workflow);
        this.updateWorkflow(clonedWorkflow, null, false);
    }

    updateWorkflow(w: Workflow, modal: ActiveModal<boolean, boolean, void>, addTrigger: boolean): void {
        this.loading = true;
        this._workflowStore.updateWorkflow(this.project.key, w).subscribe(() => {
            this.loading = false;
            this._toast.success('', this._translate.instant('workflow_updated'));
            this._workflowEventStore.unselectAll();
            if (modal) {
                modal.approve(true);
            }
        }, () => {
            if (addTrigger) {
                if (Array.isArray(this.fork.triggers) && this.fork.triggers.length) {
                    this.fork.triggers.pop();
                }
            } else {
                this.fork.name = this.previousNodeName;
            }

            this.loading = false;
        });
    }

    openDeleteForkModal(): void {
        if (this.workflow.permission !== PermissionValue.READ_WRITE_EXECUTE) {
            return;
        }
        if (this.workflowDelete) {
            this.workflowDelete.show();
        }
    }

    deleteFork(b: string): void {
        if (this.workflow.permission !== PermissionValue.READ_WRITE_EXECUTE) {
            return;
        }
        let clonedWorkflow: Workflow = cloneDeep(this.workflow);
        if (b === 'all') {
            Workflow.removeFork(clonedWorkflow, this.fork.id);
        } else if (b === 'only') {
            Workflow.removeForkWithoutChild(clonedWorkflow, this.fork.id);
        }
        this.updateWorkflow(clonedWorkflow, this.workflowDelete.modal, false);
    }
}
