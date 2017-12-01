import {Component, EventEmitter, Input, OnInit, Output} from '@angular/core';
import {Project} from '../../../../model/project.model';
import {WorkflowNode, WorkflowNodeContext} from '../../../../model/workflow.model';

@Component({
    selector: 'app-workflow-node-form',
    templateUrl: './workflow.node.form.html',
    styleUrls: ['./workflow.node.form.scss']
})
export class WorkflowNodeFormComponent {

    @Input() project: Project;
    @Input() node: WorkflowNode;
    @Output() nodeChange = new EventEmitter<WorkflowNode>();

    constructor() { }
    change(): void {
        this.node.context.application_id = Number(this.node.context.application_id);
        this.node.context.environment_id = Number(this.node.context.environment_id);
        this.node.pipeline_id = Number(this.node.pipeline_id);
        this.nodeChange.emit(this.node);
    }
}
