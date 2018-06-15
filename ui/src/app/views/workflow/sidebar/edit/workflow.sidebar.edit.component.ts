import {Component, Input} from '@angular/core';
import {Project} from '../../../../model/project.model';
import {
    Workflow,
    WorkflowNode,
    WorkflowNodeHook,
    WorkflowNodeJoin
} from '../../../../model/workflow.model';
import {AutoUnsubscribe} from '../../../../shared/decorator/autoUnsubscribe';

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
    @Input() hook: WorkflowNodeHook;
    // Flag indicate if sidebar is open
    @Input() open: boolean;

    constructor() {

    }

}
