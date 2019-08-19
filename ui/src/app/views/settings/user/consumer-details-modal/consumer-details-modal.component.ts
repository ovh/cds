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
import { Group } from 'app/model/group.model';
import { Column, ColumnType, Filter } from 'app/shared/table/data-table.component';

export enum CloseEventType {
    CHILD_DETAILS = 'CHILD_DETAILS',
    CLOSED = 'CLOSED'
}

export class CloseEvent {
    type: CloseEventType;
    payload: any;
}

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
    @Output() close = new EventEmitter<CloseEvent>();

    scopes: string;
    groups: string;
    columnsConsumer: Array<Column<AuthConsumer>>;
    filterChildren: Filter<AuthConsumer>;
    selectedChildDetails: AuthConsumer;

    constructor(
        private _modalService: SuiModalService,
        private _cd: ChangeDetectorRef
    ) {
        this.filterChildren = f => {
            const lowerFilter = f.toLowerCase();
            return (c: AuthConsumer) => {
                return c.name.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    c.description.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    c.scopes.join(' ').toLowerCase().indexOf(lowerFilter) !== -1 ||
                    (c.groups && c.groups.map(g => g.name).join(' ').toLowerCase().indexOf(lowerFilter) !== -1) ||
                    (!c.groups && lowerFilter === '*');
            }
        };

        this.columnsConsumer = [
            <Column<AuthConsumer>>{
                name: 'common_name',
                selector: (c: AuthConsumer) => c.name
            },
            <Column<AuthConsumer>>{
                name: 'common_description',
                selector: (c: AuthConsumer) => c.description
            },
            <Column<AuthConsumer>>{
                type: ColumnType.TEXT_ICONS,
                name: 'user_auth_scopes',
                selector: (c: AuthConsumer) => {
                    return {
                        value: c.scopes ? c.scopes.join(', ') : '*',
                        icons: [
                            {
                                label: 'user_auth_info_scopes',
                                class: ['info', 'circle', 'icon', 'primary', 'link'],
                                title: 'user_auth_info_scopes',
                                trigger: 'outsideClick'
                            }
                        ]
                    }
                }
            },
            <Column<AuthConsumer>>{
                name: 'user_auth_groups',
                selector: (c: AuthConsumer) => c.groups ? c.groups.map((g: Group) => g.name).join(', ') : '*'
            },
            <Column<AuthConsumer>>{
                type: ColumnType.BUTTON,
                name: 'common_action',
                class: 'two right aligned',
                selector: (c: AuthConsumer) => {
                    return {
                        title: 'common_details',
                        click: () => { this.clickConsumerDetails(c) }
                    };
                }
            }
        ];
    }

    show() {
        if (this.open) {
            return;
        }

        this.open = true;

        const config = new TemplateModalConfig<boolean, boolean, void>(this.consumerDetailsModal);
        config.mustScroll = true;
        this.modal = this._modalService.open(config);
        this.modal.onApprove(_ => { this.closeCallback() });
        this.modal.onDeny(_ => { this.closeCallback() });

        this.init();
    }

    closeCallback(): void {
        this.open = false;

        if (this.selectedChildDetails) {
            this.close.emit(<CloseEvent>{
                type: CloseEventType.CHILD_DETAILS,
                payload: this.selectedChildDetails
            });
            return;
        }

        this.close.emit(<CloseEvent>{
            type: CloseEventType.CLOSED
        });
    }

    init(): void {
        if (!this.consumer) {
            return;
        }

        this.selectedChildDetails = null;
        this.scopes = this.consumer.scopes ? this.consumer.scopes.join(', ') : '*';
        this.groups = this.consumer.groups ? this.consumer.groups.map(g => g.name).join(', ') : '*';
        this._cd.markForCheck();
    }

    clickConsumerDetails(c: AuthConsumer): void {
        this.selectedChildDetails = c;
        this.modal.approve(true);
    }
}
