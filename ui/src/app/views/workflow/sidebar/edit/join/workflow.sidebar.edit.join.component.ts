import {Component, EventEmitter, Input, ViewChild} from '@angular/core';
import {Router} from '@angular/router';
import {Workflow, WorkflowNodeJoin, WorkflowNodeJoinTrigger} from '../../../../../model/workflow.model';
import {cloneDeep} from 'lodash';
import {AutoUnsubscribe} from '../../../../../shared/decorator/autoUnsubscribe';
import {WorkflowDeleteJoinComponent} from '../../../../../shared/workflow/join/delete/workflow.join.delete.component';
import {WorkflowStore} from '../../../../../service/workflow/workflow.store';
import {WorkflowRunService} from '../../../../../service/workflow/run/workflow.run.service';
import {Project} from '../../../../../model/project.model';
import {ToastService} from '../../../../../shared/toast/ToastService';
import {TranslateService} from 'ng2-translate';
import {WorkflowTriggerJoinComponent} from '../../../../../shared/workflow/join/trigger/trigger.join.component';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {WorkflowCoreService} from '../../../../../service/workflow/workflow.core.service';
import {Subscription} from 'rxjs/Subscription';
import {finalize} from 'rxjs/operators';

@Component({
    selector: 'app-workflow-sidebar-edit-join',
    templateUrl: './workflow.sidebar.edit.join.component.html',
    styleUrls: ['./workflow.sidebar.edit.join.component.scss']
})
@AutoUnsubscribe()
export class WorkflowSidebarEditJoinComponent {

    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() join: WorkflowNodeJoin;
    @Input() readonly = false;

    disabled = false;
    loading = false;

    @ViewChild('workflowDeleteJoin')
    workflowDeleteJoin: WorkflowDeleteJoinComponent;
    @ViewChild('workflowJoinTrigger')
    workflowJoinTrigger: WorkflowTriggerJoinComponent;

    newTrigger = new WorkflowNodeJoinTrigger();

    workflowCoreSub: Subscription;

    constructor(private _workflowStore: WorkflowStore, private _toast: ToastService,
        private _translate: TranslateService, private _workflowRunService: WorkflowRunService,
        private _workflowCoreService: WorkflowCoreService, private _router: Router) {

    }

    openDeleteJoinModal(): void {
        if (this.workflowDeleteJoin) {
            this.workflowDeleteJoin.show();
        }
    }

    openTriggerJoinModal(): void {
        this.newTrigger = new WorkflowNodeJoinTrigger();
        if (this.workflowJoinTrigger) {
            this.workflowJoinTrigger.show();
        }
    }

    deleteJoin(b: boolean): void {
        console.log(this.join);
        if (b) {
            let clonedWorkflow: Workflow = cloneDeep(this.workflow);
            clonedWorkflow.joins = clonedWorkflow.joins.filter(j => j.id !== this.join.id);
            console.log(clonedWorkflow);
            this.updateWorkflow(Workflow.removeOldRef(clonedWorkflow), this.workflowDeleteJoin.modal);
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
        }, () => {
            this.loading = false;
        });
    }

    saveTrigger(): void {
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
