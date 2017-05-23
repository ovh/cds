import {Component, EventEmitter, Input, Output, ViewChild} from '@angular/core';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';
import {WorkflowNode} from '../../../../model/workflow.model';

@Component({
    selector: 'app-workflow-node-delete',
    templateUrl: './workflow.node.delete.html',
    styleUrls: ['./workflow.node.delete.scss']
})
export class WorkflowDeleteNodeComponent {

    @ViewChild('deleteModal')
    modal: SemanticModalComponent;

    @Output() deleteEvent = new EventEmitter<boolean>();
    @Input() node: WorkflowNode;

    constructor() {

    }

    show(data?: {}): void {
        this.modal.show(data);
    }

    deleteNode(): void {
        this.deleteEvent.emit(true);
    }
}
