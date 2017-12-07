import {Component, Input, OnInit} from '@angular/core';
import {AutoUnsubscribe} from '../../../../shared/decorator/autoUnsubscribe';
import {Project} from '../../../../model/project.model';
import {
    Workflow,
    WorkflowNode,
    WorkflowNodeJoin
} from '../../../../model/workflow.model';

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
    @Input() node: WorkflowNode;
    @Input() join: WorkflowNodeJoin;
    // Flag indicate if sidebar is open
    @Input() open: boolean;

    constructor() {

    }

}
