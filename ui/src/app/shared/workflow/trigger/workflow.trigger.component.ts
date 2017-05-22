import {Component, EventEmitter, Input, OnInit, Output, ViewChild} from '@angular/core';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';
import {WorkflowNode, WorkflowNodeTrigger} from '../../../model/workflow.model';
import {Project} from '../../../model/project.model';

@Component({
    selector: 'app-workflow-trigger',
    templateUrl: './workflow.trigger.html',
    styleUrls: ['./workflow.trigger.scss']
})
export class WorkflowTriggerComponent implements OnInit {

    @ViewChild('triggerModal')
    modal: SemanticModalComponent;

    @Output() triggerEvent = new EventEmitter<WorkflowNodeTrigger>();
    @Input() triggerSrcNode: WorkflowNode;
    @Input() project: Project;
    trigger: WorkflowNodeTrigger;

    constructor() {

    }

    ngOnInit() {
        this.trigger = new WorkflowNodeTrigger();
        this.trigger.workflow_node_id = this.triggerSrcNode.id;
    }

    show(data?: {}): void {
        this.modal.show(data);
    }

    hide(): void {
        this.modal.hide();
    }

    destNodeChange(node: WorkflowNode): void {
        this.trigger.workflow_dest_node = node;
    }

    saveTrigger(): void {
        this.triggerEvent.emit(this.trigger);
    }
}