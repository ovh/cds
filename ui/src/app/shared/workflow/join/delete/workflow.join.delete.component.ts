import {Component, EventEmitter, Input, Output, ViewChild} from '@angular/core';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';
import {WorkflowNodeJoin} from '../../../../model/workflow.model';

@Component({
    selector: 'app-workflow-join-delete',
    templateUrl: './workflow.join.delete.html',
    styleUrls: ['./workflow.join.delete.scss']
})
export class WorkflowDeleteJoinComponent {

    @ViewChild('deleteModal')
    modal: SemanticModalComponent;

    @Output() deleteEvent = new EventEmitter<boolean>();
    @Input() join: WorkflowNodeJoin;

    constructor() {

    }

    show(data?: {}): void {
        this.modal.show(data);
    }

    hide(): void {
        this.modal.hide();
    }

    deleteJoin(): void {
        this.deleteEvent.emit(true);
    }
}
