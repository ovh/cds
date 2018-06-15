import {Component, EventEmitter, Input, Output, ViewChild} from '@angular/core';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {WorkflowNodeJoin} from '../../../../../model/workflow.model';

@Component({
    selector: 'app-workflow-trigger-join-src',
    templateUrl: './trigger.src.html',
    styleUrls: ['./trigger.src.scss']
})
export class WorkflowJoinTriggerSrcComponent {

    @Input() join: WorkflowNodeJoin;
    @Input() loading: boolean;

    @Output() event = new EventEmitter<string>();

    @ViewChild('triggerSrcJoinModal')
    modalTemplate: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;

    constructor(private _modalService: SuiModalService) { }

    show() {
        if (this.modalTemplate) {
            const config = new TemplateModalConfig<boolean, boolean, void>(this.modalTemplate);
            this.modal = this._modalService.open(config);
        }
    }

    deleteJoin(): void {
        this.event.emit('delete_join');
    }

    deleteSrc(): void {
        this.event.emit('delete_src');
    }
}
