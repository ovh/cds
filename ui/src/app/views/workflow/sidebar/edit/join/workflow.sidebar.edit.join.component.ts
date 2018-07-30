import {Component, Input, ViewChild} from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import {cloneDeep} from 'lodash';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {Subscription} from 'rxjs/Subscription';
import {PermissionValue} from '../../../../../model/permission.model';
import {Project} from '../../../../../model/project.model';
import {Workflow, WorkflowNodeJoin, WorkflowNodeJoinTrigger} from '../../../../../model/workflow.model';
import {WorkflowEventStore} from '../../../../../service/workflow/workflow.event.store';
import {WorkflowStore} from '../../../../../service/workflow/workflow.store';
import {AutoUnsubscribe} from '../../../../../shared/decorator/autoUnsubscribe';
import {ToastService} from '../../../../../shared/toast/ToastService';
import {WorkflowDeleteJoinComponent} from '../../../../../shared/workflow/join/delete/workflow.join.delete.component';
import {WorkflowTriggerJoinComponent} from '../../../../../shared/workflow/join/trigger/trigger.join.component';

@Component({
    selector: 'app-workflow-sidebar-edit-join',
    templateUrl: './workflow.sidebar.edit.join.component.html',
    styleUrls: ['./workflow.sidebar.edit.join.component.scss']
})
@AutoUnsubscribe()
export class WorkflowSidebarEditJoinComponent {

    @Input() project: Project;
    @Input() workflow: Workflow;

    join: WorkflowNodeJoin;
    joinSub: Subscription;

    disabled = false;
    loading = false;
    permissionEnum = PermissionValue;

    @ViewChild('workflowDeleteJoin')
    workflowDeleteJoin: WorkflowDeleteJoinComponent;
    @ViewChild('workflowJoinTrigger')
    workflowJoinTrigger: WorkflowTriggerJoinComponent;

    newTrigger = new WorkflowNodeJoinTrigger();

    constructor(private _workflowStore: WorkflowStore, private _toast: ToastService, private _translate: TranslateService,
                private _workflowEventStore: WorkflowEventStore) {
        this.joinSub = this._workflowEventStore.selectedJoin().subscribe(j => this.join = j);
    }

    openDeleteJoinModal(): void {
        if (this.workflow.permission < PermissionValue.READ_WRITE_EXECUTE) {
            return;
        }
        if (this.workflowDeleteJoin) {
            this.workflowDeleteJoin.show();
        }
    }

    openTriggerJoinModal(): void {
        if (this.workflow.permission < PermissionValue.READ_WRITE_EXECUTE) {
            return;
        }
        this.newTrigger = new WorkflowNodeJoinTrigger();
        if (this.workflowJoinTrigger) {
            this.workflowJoinTrigger.show();
        }
    }

    deleteJoin(b: boolean): void {
        if (this.workflow.permission < PermissionValue.READ_WRITE_EXECUTE) {
            return;
        }
        if (b) {
            let clonedWorkflow: Workflow = cloneDeep(this.workflow);
            clonedWorkflow.joins = clonedWorkflow.joins.filter(j => j.id !== this.join.id);
            Workflow.removeOldRef(clonedWorkflow);
            this.updateWorkflow(clonedWorkflow, this.workflowDeleteJoin.modal);
        }
    }

    updateWorkflow(w: Workflow, modal?: ActiveModal<boolean, boolean, void>): void {
        this.loading = true;
        this._workflowStore.updateWorkflow(this.project.key, w).subscribe(() => {
            this.loading = false;
            this._toast.success('', this._translate.instant('workflow_updated'));
            if (modal) {
                modal.approve(true);
            }
            this._workflowEventStore.unselectAll();
        }, () => {
            this.loading = false;
        });
    }

    saveTrigger(): void {
        if (this.workflow.permission < PermissionValue.READ_WRITE_EXECUTE) {
            return;
        }
        let clonedWorkflow: Workflow = cloneDeep(this.workflow);
        let currentJoin: WorkflowNodeJoin = clonedWorkflow.joins.find(j => j.id === this.join.id);
        if (!currentJoin) {
            return;
        }

        if (!currentJoin.triggers) {
            currentJoin.triggers = new Array<WorkflowNodeJoinTrigger>();
        }
        currentJoin.triggers.push(cloneDeep(this.newTrigger));
        this.updateWorkflow(clonedWorkflow, this.workflowJoinTrigger.modal);
    }
}
