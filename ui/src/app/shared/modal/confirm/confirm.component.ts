import { ChangeDetectionStrategy, Component, EventEmitter, Input, Output, ViewChild } from '@angular/core';
import { ModalTemplate, SuiActiveModal, SuiModalService, TemplateModalConfig } from '@richardlt/ng2-semantic-ui';

@Component({
    selector: 'app-confirm-modal',
    templateUrl: './confirm.html',
    styleUrls: ['./confirm.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ConfirmModalComponent {
    @Input() title: string;
    @Input() msg: string;
    @Output() event = new EventEmitter<boolean>();

    // Ng semantic modal
    @ViewChild('myConfirmModal', {static: false})
    public myConfirmModal: ModalTemplate<boolean, boolean, void>;
    modal: SuiActiveModal<boolean, boolean, void>;
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
