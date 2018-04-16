import {Component, Input} from '@angular/core';
import {AutoUnsubscribe} from '../../../../shared/decorator/autoUnsubscribe';
import {Project} from '../../../../model/project.model';
import { Workflow} from '../../../../model/workflow.model';
import {WorkflowSidebarMode, WorkflowSidebarStore} from '../../../../service/workflow/workflow.sidebar.store';
import {Subscription} from 'rxjs/Subscription';

@Component({
    selector: 'app-workflow-sidebar-edit',
    templateUrl: './workflow.sidebar.edit.component.html',
    styleUrls: ['./workflow.sidebar.edit.component.scss']
})
@AutoUnsubscribe()
export class WorkflowSidebarEditComponent {

    // Project that contains the workflow
    @Input() project: Project;
    @Input() workflow: Workflow;

    mode: string;
    modes = WorkflowSidebarMode;

    subs: Subscription;

    constructor(private _workflowSidebarStore: WorkflowSidebarStore) {
        this.subs = this._workflowSidebarStore.sidebarMode().subscribe(m => {
            this.mode = m;
        });
    }

}
