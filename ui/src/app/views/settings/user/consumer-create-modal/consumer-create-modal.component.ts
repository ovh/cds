import {
    ChangeDetectionStrategy,
    Component,
    EventEmitter,
    Output,
    ViewChild
} from '@angular/core';
import { ModalTemplate, SuiActiveModal, SuiModalService, TemplateModalConfig } from '@richardlt/ng2-semantic-ui';
import { AuthConsumer } from 'app/model/authentication.model';

@Component({
    selector: 'app-consumer-create-modal',
    templateUrl: './consumer-create-modal.html',
    styleUrls: ['./consumer-create-modal.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ConsumerCreateModalComponent {
    @ViewChild('consumerCreateModal', { static: false }) consumerDetailsModal: ModalTemplate<boolean, boolean, void>;
    modal: SuiActiveModal<boolean, boolean, void>;
    open: boolean;

    @Output() close = new EventEmitter();

    newConsumer: AuthConsumer = new AuthConsumer();

    constructor(
        private _modalService: SuiModalService
    ) { }

    show() {
        if (this.open) {
            return;
        }

        this.open = true;

        const config = new TemplateModalConfig<boolean, boolean, void>(this.consumerDetailsModal);
        config.mustScroll = true;
        this.modal = this._modalService.open(config);
        this.modal.onApprove(() => {
            this.open = false;
            this.close.emit();
        });
        this.modal.onDeny(() => {
            this.open = false;
            this.close.emit();
        });

        this.init();
    }

    clickClose() {
        this.modal.approve(true);
    }

    init(): void {
        this.newConsumer = new AuthConsumer();
    }
}
