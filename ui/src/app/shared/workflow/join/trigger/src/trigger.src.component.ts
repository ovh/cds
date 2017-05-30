import {Component, EventEmitter, Input, Output, ViewChild} from '@angular/core';
import {WorkflowNodeJoin} from '../../../../../model/workflow.model';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';

@Component({
    selector: 'app-workflow-trigger-join-src',
    templateUrl: './trigger.src.html',
    styleUrls: ['./trigger.src.scss']
})
export class WorkflowJoinTriggerSrcComponent {

    @Input() join: WorkflowNodeJoin;

    @Output() event = new EventEmitter<string>();

    @ViewChild('triggerSrcJoinModal')
    modal: SemanticModalComponent;

    constructor() { }

    show(data: { observable: boolean; closable: boolean; autofocus: boolean }) {
        if (this.modal) {
            this.modal.show(data);
        }
    }

    hide(): void {
        if (this.modal) {
            this.modal.hide();
        }
    }

    deleteJoin(): void {
        this.event.emit('delete_join');
    }

    deleteSrc(): void {
        this.event.emit('delete_src');
    }
}
