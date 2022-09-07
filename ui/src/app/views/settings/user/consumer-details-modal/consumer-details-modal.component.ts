import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    Input, OnInit,
} from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { AuthConsumer, AuthConsumerValidityPeriod, AuthSession } from 'app/model/authentication.model';
import { Group } from 'app/model/group.model';
import { AuthentifiedUser, AuthSummary } from 'app/model/user.model';
import { AuthenticationService } from 'app/service/authentication/authentication.service';
import { UserService } from 'app/service/user/user.service';
import { Item } from 'app/shared/menu/menu.component';
import { Column, ColumnType, Filter } from 'app/shared/table/data-table.component';
import { ToastService } from 'app/shared/toast/ToastService';
import { AuthenticationState } from 'app/store/authentication.state';
import * as moment from 'moment';
import { NzModalRef } from 'ng-zorro-antd/modal';

export enum CloseEventType {
    CHILD_DETAILS = 'CHILD_DETAILS',
    DELETE_OR_DETACH = 'DELETE_OR_DETACH',
    CLOSED = 'CLOSED',
    REGEN = 'REGEN'
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
export class ConsumerDetailsModalComponent implements OnInit {

    @Input() user: AuthentifiedUser;
    @Input() consumer: AuthConsumer;

    loading: boolean;
    currentAuthSummary: AuthSummary;
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
    regenConsumerSigninToken: string;
    warningText: string;
    columnsValidityPeriods: Array<Column<AuthConsumerValidityPeriod>>;

    constructor(
        private _modal: NzModalRef,
        private _authenticationService: AuthenticationService,
        private _userService: UserService,
        private _cd: ChangeDetectorRef,
        private _toast: ToastService,
        private _store: Store,
        private _translate: TranslateService
    ) {
        this.currentAuthSummary = this._store.selectSnapshot(AuthenticationState.summary);

        this.menuItems = [].concat(defaultMenuItems);

        this.filterChildren = f => {
            const lowerFilter = f.toLowerCase();
            return (c: AuthConsumer) => c.name.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    c.description.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    c.id.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    c.scope_details.map(s => s.scope).join(' ').toLowerCase().indexOf(lowerFilter) !== -1 ||
                    (c.groups && c.groups.map(g => g.name).join(' ').toLowerCase().indexOf(lowerFilter) !== -1) ||
                    (!c.groups && lowerFilter === '*');
        };

        this.columnsConsumers = [
            <Column<AuthConsumer>>{
                type: ColumnType.TEXT_LABELS,
                name: 'common_name',
                selector: (c: AuthConsumer) => {
                    let labels = [];

                    if (c.disabled) {
                        labels.push({ color: 'error', title: 'user_auth_consumer_disabled' });
                    }

                    return {
                        value: c.name,
                        labels
                    };
                }
            },
            <Column<AuthConsumer>>{
                name: 'user_auth_scopes',
                selector: (c: AuthConsumer) => c.scope_details ? c.scope_details.map(s => s.scope).join(', ') : '*'
            },
            <Column<AuthConsumer>>{
                name: 'user_auth_groups',
                selector: (c: AuthConsumer) => c.groups ? c.groups.map((g: Group) => g.name).join(', ') : '*'
            },
            <Column<AuthConsumer>>{
                type: ColumnType.BUTTON,
                name: 'Action',
                class: 'rightAlign',
                selector: (c: AuthConsumer) => ({
                        title: 'Details',
                        buttonDanger: false,
                        click: () => this.clickConsumerDetails(c)
                    })
            }
        ];

        this.filterSessions = f => {
            const lowerFilter = f.toLowerCase();
            return (s: AuthSession) => s.consumer.name.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    s.id.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    s.consumer_id.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    s.created.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    s.expire_at.toLowerCase().indexOf(lowerFilter) !== -1;
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
                    };
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

        this.columnsValidityPeriods = [
            <Column<AuthConsumerValidityPeriod>>{
                type: ColumnType.DATE,
                name: 'Issued at',
                selector: (s: AuthConsumerValidityPeriod) => moment(s.issued_at).format()
            },
            <Column<AuthConsumerValidityPeriod>>{
                type: ColumnType.TEXT_LABELS,
                name: 'Duration',
                selector: (s: AuthConsumerValidityPeriod) => {
                    let labels = [];
                    if (s.duration === 0) {
                        return {value: '-', labels};
                    }

                    let ms = (s.duration / 1000000);
                    let days = Math.floor(ms / (1000 * 60 * 60 * 24));
                    let unit = ' days';
                    if (days <= 1) {
                        unit = ' day';
                    }
                    return {
                        value: days + unit,
                        labels
                    };
                }
            }
        ];
    }

    ngOnInit(): void {
        if (!this.consumer) {
            return;
        }
        this.regenConsumerSigninToken = null;

        this.selectedChildDetails = null;
        this.consumerDeletedOrDetached = false;
        this.scopes = this.consumer.scope_details ? this.consumer.scope_details.map(s => s.scope).join(', ') : '*';
        this.groups = this.consumer.groups ? this.consumer.groups.map(g => g.name).join(', ') : '*';

        if (this.consumer.warnings && this.consumer.warnings.length > 0) {
            this.warningText = this.consumer.warnings.map(w => {
                switch (w.type) {
                    case 'last-group-removed':
                        return this._translate.instant('user_auth_consumer_warning_last_group_removed', { name: w.group_name });
                    case 'group-invalid':
                        return this._translate.instant('user_auth_consumer_warning_group_invalid', { name: w.group_name });
                    case 'group-removed':
                        return this._translate.instant('user_auth_consumer_warning_group_removed', { name: w.group_name });
                }
                return w.type;
            }).join(' ');
        }

        this.menuItems = [].concat(defaultMenuItems);
        if (this.consumer.parent) {
            this.menuItems.push(<Item>{
                translate: 'auth_consumer_details_parent',
                key: 'parent'
            });
        }
        if (this.consumer.children.length > 0) {
            this.menuItems.push(<Item>{
                translate: 'auth_consumer_details_children',
                key: 'children'
            });
        }
        if (this.consumer.validity_periods.length > 0) {
            this.menuItems.push(<Item>{
                translate: 'validity_periods',
                key: 'validity_periods'
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
        this._modal.close(<CloseEvent>{
            type: CloseEventType.CHILD_DETAILS,
            payload: this.selectedChildDetails
        });
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
            this._modal.close(<CloseEvent>{
                type: CloseEventType.DELETE_OR_DETACH,
                payload: this.consumer.id
            })
        });
    }

    clickDetach(): void {
        this.loading = true;
        this._authenticationService.detach(this.consumer.type).subscribe(() => {
            this.consumerDeletedOrDetached = true;
            this.loading = false;
            this._modal.close(<CloseEvent>{
                type: CloseEventType.DELETE_OR_DETACH,
                payload: this.consumer.id
            });
        });
    }

    clickClose(): void {
        if (this.regenConsumerSigninToken) {
            this._modal.close (<CloseEvent>{
                type: CloseEventType.REGEN,
                payload: this.consumer.id
            });
            return;
        }
        this._modal.close(<CloseEvent>{
            type: CloseEventType.CLOSED,
            payload: this.consumer.id
        })
    }

    clickRegen(revokeSession: boolean): void {
        this.regenConsumerSigninToken = null;
        this._userService.regenConsumer(this.user.username, this.consumer, revokeSession).subscribe(res => {
            this.consumer = res.consumer;
            this.regenConsumerSigninToken = res.token;
            this._cd.markForCheck();
        });
    }
}
