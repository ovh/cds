import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    Output,
    ViewChild
} from '@angular/core';
import { ModalTemplate, SuiActiveModal, SuiModalService, TemplateModalConfig } from '@richardlt/ng2-semantic-ui';
import { AuthConsumer } from 'app/model/authentication.model';

@Component({
    selector: 'app-consumer-details-modal',
    templateUrl: './consumer-details-modal.html',
    styleUrls: ['./consumer-details-modal.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ConsumerDetailsModalComponent {
    @ViewChild('consumerDetailsModal', { static: false }) consumerDetailsModal: ModalTemplate<boolean, boolean, void>;
    modal: SuiActiveModal<boolean, boolean, void>;
    open: boolean;

    @Input() consumer: AuthConsumer;
    @Output() close = new EventEmitter();

    scopes: string;
    groups: string;

    constructor(
        private _modalService: SuiModalService,
        private _cd: ChangeDetectorRef
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

    init(): void {
        if (!this.consumer) {
            return;
        }

        this.scopes = this.consumer.scopes ? this.consumer.scopes.join(', ') : '*';
        this.groups = this.consumer.groups ? this.consumer.groups.map(g => g.name).join(', ') : '*';
        this._cd.markForCheck();
    }

    clickClose() {
        this.modal.approve(true);
    }
}
