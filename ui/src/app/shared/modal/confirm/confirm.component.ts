import { Component, EventEmitter, Input, Output, ViewChild } from '@angular/core';
import { ModalTemplate, SuiModalService, TemplateModalConfig } from 'ng2-semantic-ui';
import { ActiveModal } from 'ng2-semantic-ui/dist';

@Component({
    selector: 'app-confirm-modal',
    templateUrl: './confirm.html',
    styleUrls: ['./confirm.scss']
})
export class ConfirmModalComponent {
    @Input() title: string;
    @Input() msg: string;
    @Output() event = new EventEmitter<boolean>();

    // Ng semantic modal
    @ViewChild('myConfirmModal')
    public myConfirmModal: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;
    modalConfig: TemplateModalConfig<boolean, boolean, void>;

    constructor(private _modalService: SuiModalService) { }

    show() {
        this.modalConfig = new TemplateModalConfig<boolean, boolean, void>(this.myConfirmModal);
        this.modal = this._modalService.open(this.modalConfig);
    }

    eventAndClose(confirm: boolean) {
        this.event.emit(confirm);
        this.modal.approve(true);
    }
}
