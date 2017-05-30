import {Component, EventEmitter, Input, Output, ViewChild} from '@angular/core';
import {Project} from '../../../../model/project.model';
import {WorkflowNode} from '../../../../model/workflow.model';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';

@Component({
    selector: 'app-workflow-node-context',
    templateUrl: './node.context.html',
    styleUrls: ['./node.context.scss']
})
export class WorkflowNodeContextComponent {

    @Input() project: Project;
    @Input() node: WorkflowNode;

    @ViewChild('nodeContextModal')
    nodeContextModal: SemanticModalComponent;

    @Output() contextEvent = new EventEmitter<WorkflowNode>();

    constructor() { }

    show(data?: {}): void {
        if (this.nodeContextModal) {
            this.nodeContextModal.show(data);
        }
    }

    hide(): void {
        if (this.nodeContextModal) {
            this.nodeContextModal.hide();
        }
    }

    saveContext(): void {
        this.contextEvent.emit(this.node);
    }
}
