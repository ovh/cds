import {Component, EventEmitter, Input, Output, ViewChild} from '@angular/core';
import {WorkflowNodeJoin} from '../../../../model/workflow.model';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';

@Component({
    selector: 'app-workflow-join-delete',
    templateUrl: './workflow.join.delete.html',
    styleUrls: ['./workflow.join.delete.scss']
})
export class WorkflowDeleteJoinComponent {

    @ViewChild('deleteModal')
    modalTemplate: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;

    @Output() deleteEvent = new EventEmitter<boolean>();
    @Input() join: WorkflowNodeJoin;
    @Input() loading: boolean;

    constructor(private _modalService: SuiModalService) {

    }

    show(): void {
        const config = new TemplateModalConfig<boolean, boolean, void>(this.modalTemplate);
        this.modal = this._modalService.open(config);
    }

    deleteJoin(): void {
        this.deleteEvent.emit(true);
    }
}
