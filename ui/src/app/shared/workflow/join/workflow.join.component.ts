import {AfterViewInit, Component, ElementRef, Input, ViewChild} from '@angular/core';
import {Workflow, WorkflowNodeJoin} from '../../../model/workflow.model';
import {cloneDeep} from 'lodash';
import {WorkflowDeleteJoinComponent} from './delete/workflow.join.delete.component';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';
import {WorkflowStore} from '../../../service/workflow/workflow.store';
import {Project} from '../../../model/project.model';
import {ToastService} from '../../toast/ToastService';
import {TranslateService} from 'ng2-translate';

@Component({
    selector: 'app-workflow-join',
    templateUrl: './workflow.join.html',
    styleUrls: ['./workflow.join.scss']
})
export class WorkflowJoinComponent implements AfterViewInit{

    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() join: WorkflowNodeJoin;

    @ViewChild('workflowDeleteJoin')
    workflowDeleteJoin: WorkflowDeleteJoinComponent;

    constructor(private elementRef: ElementRef, private _workflowStore: WorkflowStore, private _toast: ToastService,
        private _translate: TranslateService) { }

    ngAfterViewInit() {
        this.elementRef.nativeElement.style.position = 'fixed';
        this.elementRef.nativeElement.style.top = 0;
    }

    openDeleteJoinModal(): void {
        if (this.workflowDeleteJoin) {
            this.workflowDeleteJoin.show({observable: true, closable: false, autofocus: false});
        }
    }

    deleteJoin(b: boolean): void {
        if (b) {
            let clonedWorkflow: Workflow = cloneDeep(this.workflow);
            clonedWorkflow.joins = clonedWorkflow.joins.filter(j => j.id !== this.join.id);
            this.updateWorkflow(clonedWorkflow, this.workflowDeleteJoin.modal);
        }
    }

    updateWorkflow(w: Workflow, modal?: SemanticModalComponent): void {
        this._workflowStore.updateWorkflow(this.project.key, w).subscribe(() => {
            this._toast.success('', this._translate.instant('workflow_updated'));
            if (modal) {
                modal.hide();
            }
        });
    }
}