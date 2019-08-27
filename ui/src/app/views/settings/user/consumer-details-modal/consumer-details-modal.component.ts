import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    Output,
    ViewChild
} from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { ModalTemplate, SuiActiveModal, SuiModalService, TemplateModalConfig } from '@richardlt/ng2-semantic-ui';
import { AuthConsumer, AuthSession } from 'app/model/authentication.model';
import { Group } from 'app/model/group.model';
import { AuthentifiedUser } from 'app/model/user.model';
import { AuthenticationService } from 'app/service/authentication/authentication.service';
import { UserService } from 'app/service/services.module';
import { Item } from 'app/shared/menu/menu.component';
import { Column, ColumnType, Filter } from 'app/shared/table/data-table.component';
import { ToastService } from 'app/shared/toast/ToastService';

export enum CloseEventType {
    CHILD_DETAILS = 'CHILD_DETAILS',
    DELETE_OR_DETACH = 'DELETE_OR_DETACH',
    CLOSED = 'CLOSED'
}

export class CloseEvent {
    type: CloseEventType;
    payload: any;
}

const defaultMenuItems = [<Item>{
    translate: 'user_auth_sessions',
    key: 'sessions',
    default: true
}];

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

    @Input() user: AuthentifiedUser;
    @Input() consumer: AuthConsumer;
    @Output() close = new EventEmitter<CloseEvent>();

    scopes: string;
    groups: string;
    columnsConsumers: Array<Column<AuthConsumer>>;
    filterChildren: Filter<AuthConsumer>;
    selectedChildDetails: AuthConsumer;
    menuItems: Array<Item>;
    selectedItem: Item;
    columnsSessions: Array<Column<AuthSession>>;
    filterSessions: Filter<AuthSession>;
    consumerDeletedOrDetached: boolean;
    regenConsumer: AuthConsumer;
    regenConsumerSigninToken: string;

    constructor(
        private _modalService: SuiModalService,
        private _authenticationService: AuthenticationService,
        private _userService: UserService,
        private _cd: ChangeDetectorRef,
        private _toast: ToastService,
        private _translate: TranslateService
    ) {
        this.menuItems = [].concat(defaultMenuItems);

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

        this.columnsConsumers = [
            <Column<AuthConsumer>>{
                name: 'common_name',
                selector: (c: AuthConsumer) => c.name
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

        this.filterSessions = f => {
            const lowerFilter = f.toLowerCase();
            return (s: AuthSession) => {
                return s.consumer.name.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    s.consumer_id.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    s.created.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    s.expire_at.toLowerCase().indexOf(lowerFilter) !== -1;
            }
        };

        this.columnsSessions = [
            <Column<AuthSession>>{
                type: ColumnType.TEXT_LABELS,
                name: 'common_id',
                selector: (s: AuthSession) => {
                    let labels = [];

                    if (s.current) {
                        labels.push({ color: 'blue', title: 'user_auth_session_current' });
                    }

                    return {
                        value: s.id,
                        labels
                    }
                }
            },
            <Column<AuthSession>>{
                type: ColumnType.DATE,
                name: 'common_created',
                selector: (s: AuthSession) => s.created
            },
            <Column<AuthSession>>{
                type: ColumnType.DATE,
                name: 'user_auth_expire_at',
                selector: (s: AuthSession) => s.expire_at
            }
        ];
    }

    show() {
        if (this.open) {
            return;
        }

        this.open = true;
        this.regenConsumer = undefined;
        this.regenConsumerSigninToken = undefined;
        const config = new TemplateModalConfig<boolean, boolean, void>(this.consumerDetailsModal);
        config.mustScroll = true;
        this.modal = this._modalService.open(config);
        this.modal.onApprove(_ => { this.closeCallback() });
        this.modal.onDeny(_ => { this.closeCallback() });

        this.init();
    }

    closeCallback(): void {
        this.open = false;

        if (this.consumerDeletedOrDetached) {
            this.close.emit(<CloseEvent>{
                type: CloseEventType.DELETE_OR_DETACH,
                payload: this.consumer.id
            });
            return;
        }

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
        this.consumerDeletedOrDetached = false;
        this.scopes = this.consumer.scopes ? this.consumer.scopes.join(', ') : '*';
        this.groups = this.consumer.groups ? this.consumer.groups.map(g => g.name).join(', ') : '*';

        this.menuItems = [].concat(defaultMenuItems);
        if (this.consumer.parent) {
            this.menuItems.push(<Item>{
                translate: 'auth_consumer_details_parent',
                key: 'parent'
            })
        }
        if (this.consumer.children.length > 0) {
            this.menuItems.push(<Item>{
                translate: 'auth_consumer_details_children',
                key: 'children'
            });
        }
        this._cd.markForCheck();
    }

    selectMenuItem(item: Item): void {
        this.selectedItem = item;
        this._cd.markForCheck();
    }

    clickConsumerDetails(c: AuthConsumer): void {
        this.selectedChildDetails = c;
        this.modal.approve(true);
    }

    clickResetPassword(): void {
        this._authenticationService.localAskReset()
            .subscribe(() => {
                this._toast.success('', this._translate.instant('auth_ask_reset_success'));
            });
    }

    clickDelete(): void {
        this._userService.deleteConsumer(this.user.username, this.consumer).subscribe(() => {
            this.consumerDeletedOrDetached = true;
            this.modal.approve(true);
        });
    }

    clickDetach(): void {
        this._authenticationService.detach(this.consumer.type).subscribe(() => {
            this.consumerDeletedOrDetached = true;
            this.modal.approve(true);
        });
    }

    clickClose(): void {
        this.modal.approve(true);
    }

    clickRegen(revokeSession: boolean): void {
        this.regenConsumer = undefined;
        this.regenConsumerSigninToken = undefined;
        this._authenticationService.builtinRegen(this.consumer.id, revokeSession).subscribe(response => {
            this.regenConsumerSigninToken = response.token;
            this.regenConsumer = response.consumer;
        });
    }
}
