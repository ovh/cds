import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnInit, ViewChild } from '@angular/core';
import { NgForm } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Transition, TransitionController, TransitionDirection } from '@richardlt/ng2-semantic-ui';
import { AuthConsumer, AuthDriverManifest, AuthDriverManifests, AuthSession } from 'app/model/authentication.model';
import { Group } from 'app/model/group.model';
import { AuthentifiedUser, AuthSummary, UserContact } from 'app/model/user.model';
import { AuthenticationService } from 'app/service/authentication/authentication.service';
import { UserService } from 'app/service/user/user.service';
import { PathItem } from 'app/shared/breadcrumb/breadcrumb.component';
import { Item } from 'app/shared/menu/menu.component';
import { Column, ColumnType, Filter } from 'app/shared/table/data-table.component';
import { ToastService } from 'app/shared/toast/ToastService';
import { AuthenticationState } from 'app/store/authentication.state';
import { forkJoin } from 'rxjs';
import { finalize } from 'rxjs/operators';
import { CloseEventType, ConsumerCreateModalComponent } from '../consumer-create-modal/consumer-create-modal.component';
import {
    CloseEvent,
    CloseEventType as DetailsCloseEventType,
    ConsumerDetailsModalComponent
} from '../consumer-details-modal/consumer-details-modal.component';

const defaultMenuItems = [<Item>{
    translate: 'user_profile_btn',
    key: 'profile',
    default: true
}, <Item>{
    translate: 'user_groups_btn',
    key: 'groups'
}];

const usernamePattern = new RegExp('^[a-zA-Z0-9._-]{1,}$');

@Component({
    selector: 'app-user-edit',
    templateUrl: './user.edit.html',
    styleUrls: ['./user.edit.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class UserEditComponent implements OnInit {
    transitionController = new TransitionController();

    @ViewChild('consumerDetailsModal')
    consumerDetailsModal: ConsumerDetailsModalComponent;

    @ViewChild('consumerCreateModal')
    consumerCreateModal: ConsumerCreateModalComponent;

    @ViewChild('ldapSigninForm')
    ldapSigninForm: NgForm;

    loading = false;
    deleteLoading = false;
    groupsAdmin: Array<Group>;
    userPatternError = false;
    username: string;
    currentAuthSummary: AuthSummary;
    editable: boolean;
    path: Array<PathItem>;
    menuItems: Array<Item>;
    selectedItem: Item;
    loadingUser: boolean;
    user: AuthentifiedUser;
    columnsGroups: Array<Column<Group>>;
    loadingGroups = false;
    groups: Array<Group>;
    columnsContacts: Array<Column<UserContact>>;
    loadingContacts = false;
    contacts: Array<UserContact>;
    loadingAuthData: boolean;
    drivers: Array<AuthDriverManifest>;
    consumers: Array<AuthConsumer>;
    mConsumers: { [key: string]: AuthConsumer };
    myConsumers: Array<AuthConsumer>;
    selectedConsumer: AuthConsumer;
    columnsConsumers: Array<Column<AuthConsumer>>;
    filterConsumers: Filter<AuthConsumer>;
    columnsSessions: Array<Column<AuthSession>>;
    filterSessions: Filter<AuthSession>;
    sessions: Array<AuthSession>;
    loadingLocalReset: boolean;
    showLDAPSigninForm: boolean;

    constructor(
        private _authenticationService: AuthenticationService,
        private _userService: UserService,
        private _translate: TranslateService,
        private _route: ActivatedRoute,
        private _router: Router,
        private _store: Store,
        private _toast: ToastService,
        private _cd: ChangeDetectorRef
    ) {
        this.currentAuthSummary = this._store.selectSnapshot(AuthenticationState.summary);

        this.menuItems = [].concat(defaultMenuItems);

        this.columnsGroups = [
            <Column<Group>>{
                type: ColumnType.ROUTER_LINK,
                name: 'common_name',
                selector: (g: Group) => ({
                    link: '/settings/group/' + g.name,
                    value: g.name
                })
            },
            <Column<Group>>{
                name: 'user_group_role',
                selector: (g: Group) => g.admin ? this._translate.instant('user_group_admin') : this._translate.instant('user_group_member')
            },
        ];

        this.columnsContacts = [
            <Column<UserContact>>{
                name: 'common_name',
                class: 'two',
                selector: (c: UserContact) => c.type
            },
            <Column<UserContact>>{
                type: ColumnType.TEXT_LABELS,
                class: 'fourteen',
                name: 'common_value',
                selector: (c: UserContact) => {
                    let labels = [];

                    if (c.primary) {
                        labels.push({ color: 'green', title: 'user_contact_primary' });
                    }
                    if (!c.verified) {
                        labels.push({ color: 'red', title: 'user_contact_not_verified' });
                    }

                    return {
                        value: c.value,
                        labels
                    }
                }
            }
        ];

        this.filterConsumers = f => {
            const lowerFilter = f.toLowerCase();
            return (c: AuthConsumer) => c.name.toLowerCase().indexOf(lowerFilter) !== -1 ||
                c.description.toLowerCase().indexOf(lowerFilter) !== -1 ||
                c.id.toLowerCase().indexOf(lowerFilter) !== -1 ||
                c.scope_details.map(s => s.scope).join(' ').toLowerCase().indexOf(lowerFilter) !== -1 ||
                (c.groups && c.groups.map(g => g.name).join(' ').toLowerCase().indexOf(lowerFilter) !== -1) ||
                (!c.groups && lowerFilter === '*')
        };

        this.columnsConsumers = [
            <Column<AuthConsumer>>{
                type: ColumnType.TEXT_LABELS,
                name: 'common_name',
                selector: (c: AuthConsumer) => {
                    let labels = [];

                    if (c.disabled) {
                        labels.push({ color: 'red', title: 'user_auth_consumer_disabled' });
                    }

                    return {
                        value: c.name,
                        labels
                    }
                }
            },
            <Column<AuthConsumer>>{
                name: 'common_description',
                selector: (c: AuthConsumer) => c.description
            },
            <Column<AuthConsumer>>{
                name: 'user_auth_scopes',
                selector: (c: AuthConsumer) => c.scope_details.map(s => s.scope).join(', ')
            },
            <Column<AuthConsumer>>{
                type: ColumnType.TEXT_ICONS,
                name: 'user_auth_groups',
                selector: (c: AuthConsumer) => {
                    let icons = [];

                    if (c.warnings && c.warnings.length > 0) {
                        let text = c.warnings.map(w => {
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

                        icons.push({
                            label: text,
                            class: ['info', 'exclamation', 'triangle', 'icon', 'yellow', 'link'],
                            title: text
                        });
                    }

                    return {
                        value: c.groups ? c.groups.map((g: Group) => g.name).join(', ') : '*',
                        icons
                    }
                }
            },
            <Column<AuthConsumer>>{
                type: ColumnType.BUTTON,
                name: 'common_action',
                class: 'two right aligned',
                selector: (c: AuthConsumer) => ({
                    title: 'common_details',
                    click: () => {
                        this.clickConsumerDetails(c)
                    }
                })
            }
        ];

        this.filterSessions = f => {
            const lowerFilter = f.toLowerCase();
            return (s: AuthSession) => s.consumer.name.toLowerCase().indexOf(lowerFilter) !== -1 ||
                s.id.toLowerCase().indexOf(lowerFilter) !== -1 ||
                s.consumer_id.toLowerCase().indexOf(lowerFilter) !== -1 ||
                s.created.toLowerCase().indexOf(lowerFilter) !== -1 ||
                s.expire_at.toLowerCase().indexOf(lowerFilter) !== -1
        };

        this.columnsSessions = [
            <Column<AuthSession>>{
                type: ColumnType.TEXT_ICONS,
                name: 'user_auth_consumer',
                selector: (s: AuthSession) => {
                    let icons = [];

                    if (this.mConsumers && this.mConsumers[s.consumer_id]) {
                        let consumer = this.mConsumers[s.consumer_id];

                        let icon = {
                            label: `ID: ${s.consumer_id}`,
                            title: `ID: ${s.consumer_id}`
                        };
                        switch (consumer.type) {
                            case 'builtin':
                                icon['class'] = ['info', 'circle', 'icon', 'link'];
                                break;
                            case 'local':
                                icon['class'] = ['lock', 'icon'];
                                break;
                            case 'ldap':
                                icon['class'] = ['address', 'book', 'icon'];
                                break;
                            case 'corporate-sso':
                                icon['class'] = ['shield', 'alternate', 'icon'];
                                break;
                            case 'openid-connect':
                                icon['class'] = ['openid', 'icon'];
                                break;
                            default:
                                icon['class'] = [consumer.type, 'icon'];
                                break;
                        }

                        icons.push(icon);
                    }

                    if (s.mfa) {
                        const lastActivity = s.last_activity ? `Last activity: ${s.last_activity}.` : 'Expired.';
                        icons.push({
                            label: `MFA. ${lastActivity}`,
                            title: `MFA. ${lastActivity}`,
                            class: ['key', 'icon']
                        });
                    }

                    return {
                        value: s.consumer.name,
                        icons
                    };
                }
            },
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
            },
            <Column<AuthSession>>{
                type: ColumnType.CONFIRM_BUTTON,
                name: 'common_action',
                class: 'two right aligned',
                disabled: true,
                selector: (s: AuthSession) => ({
                    title: 'user_auth_revoke_btn',
                    color: 'red',
                    click: () => {
                        this.clickSessionRevoke(s)
                    }
                })
            }
        ];
    }

    ngOnInit() {
        this._route.params.subscribe(params => {
            this.username = params['username'];
            this.getUser();
        });
    }

    clickConsumerDetails(selected: AuthConsumer): void {
        this.selectedConsumer = selected;

        // calculate children for selected consumer
        this.selectedConsumer.children = this.consumers.filter(c => c.parent_id === this.selectedConsumer.id);
        this.selectedConsumer.sessions = this.sessions.filter(s => s.consumer_id === this.selectedConsumer.id);

        this._cd.detectChanges(); // manually ask for detect changes to allow modal data to be set before opening
        this.consumerDetailsModal.show();
    }

    clickConsumerLocalReset(): void {
        this.loadingLocalReset = true;
        this._cd.markForCheck();
        this._authenticationService.localAskReset()
            .pipe(finalize(() => {
                this.loadingLocalReset = false;
                this._cd.markForCheck();
            }))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('auth_ask_reset_success'));
            });
    }

    clickConsumerLDAPSignin(): void {
        if (this.showLDAPSigninForm) {
            this._authenticationService.ldapSignin(this.ldapSigninForm.value.bind, this.ldapSigninForm.value.password).subscribe(() => {
                this.showLDAPSigninForm = false;
                this._cd.markForCheck();
                this.getAuthData();
            });
            return;
        }

        this.transitionController.animate(
            new Transition('scale', 150, TransitionDirection.Out, () => {
                this.showLDAPSigninForm = true;
                this._cd.detectChanges();
                this.transitionController.animate(
                    new Transition('scale', 150, TransitionDirection.In, () => { })
                );
            })
        );
    }

    clickConsumerDetach(c: AuthConsumer): void {
        let callback: () => any;

        let currentSession = this.sessions.find(s => s.current);
        if (currentSession && currentSession.consumer_id === c.id) {
            callback = () => {
                this._router.navigate(['/auth/signin']);
            };
        } else {
            callback = () => {
                this.getAuthData();
            };
        }

        this._authenticationService.detach(c.type).subscribe(callback);
    }

    clickConsumerCreate(): void {
        this.consumerCreateModal.show();
    }

    clickSessionRevoke(s: AuthSession): void {
        if (s.current) {
            this._authenticationService.signout().subscribe(() => {

            });
        } else {
            this._userService.deleteSession(this.currentAuthSummary.user.username, s.id).subscribe(() => {
                this.getAuthData();
            });
        }
    }

    clickDelete(): void {
        this.deleteLoading = true;
        this._cd.markForCheck();
        this._userService.delete(this.username)
            .pipe(finalize(() => {
                this.deleteLoading = false;
                this._cd.markForCheck();
            }))
            .subscribe(_ => {
                this._toast.success('', this._translate.instant('user_deleted'));
                this._router.navigate(['../'], { relativeTo: this._route });
            });
    }

    clickSave(): void {
        this.userPatternError = false;
        if (!this.user.username || !this.user.fullname) {
            return;
        }
        if (!usernamePattern.test(this.user.username)) {
            this.userPatternError = true;
            this._cd.markForCheck();
            return;
        }

        this.loading = true;
        this._cd.markForCheck();
        this._userService.update(this.username, this.user)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(u => {
                this._toast.success('', this._translate.instant('user_saved'));
                this.user = u;
                this.setDataFromUser();
                this.updatePath();
                this._router.navigate(['/settings', 'user', this.user.username], { relativeTo: this._route });
            });
    }

    updatePath(): void {
        this.path = [<PathItem>{
            translate: 'common_settings'
        }, <PathItem>{
            translate: 'user_list_title',
            routerLink: ['/', 'settings', 'user']
        }, <PathItem>{
            text: this.user.username,
            routerLink: ['/', 'settings', 'user', this.user.username]
        }];
    }

    selectMenuItem(item: Item): void {
        switch (item.key) {
            case 'groups':
                this.getGroups();
                break;
            case 'contacts':
                this.getContacts();
                break;
            case 'authentication':
                this.getAuthData();
                break;
        }
        this.selectedItem = item;
        this._cd.markForCheck();
    }

    getUser(): void {
        this.loadingUser = true;
        this._cd.markForCheck();
        this._userService.get(this.username)
            .pipe(finalize(() => {
                this.loadingUser = false;
                this._cd.markForCheck();
            }))
            .subscribe(u => {
                this.user = u;
                this.setDataFromUser();
                this.updatePath();
            });
    }

    setDataFromUser(): void {
        this.editable = this.user.id === this.currentAuthSummary.user.id || this.currentAuthSummary.isAdmin();

        if (this.user.id === this.currentAuthSummary.user.id || this.currentAuthSummary.isMaintainer()) {
            this.menuItems = defaultMenuItems.concat([<Item>{
                translate: 'user_contacts_btn',
                key: 'contacts'
            }, <Item>{
                translate: 'user_authentication_btn',
                key: 'authentication'
            }]);
        } else {
            this.menuItems = [].concat(defaultMenuItems);
        }

        // Enable revoke session button only if editable
        this.columnsSessions[4].disabled = !this.editable;
    }

    getGroups(): void {
        this.loadingGroups = true;
        this._cd.markForCheck();
        this._userService.getGroups(this.username)
            .pipe(finalize(() => {
                this.loadingGroups = false;
                this._cd.markForCheck();
            }))
            .subscribe((gs) => {
                this.groups = gs;
            });
    }

    getContacts(): void {
        this.loadingContacts = true;
        this._cd.markForCheck();
        this._userService.getContacts(this.username)
            .pipe(finalize(() => {
                this.loadingContacts = false;
                this._cd.markForCheck();
            }))
            .subscribe((cs) => {
                this.contacts = cs;
            });
    }

    getAuthData(): void {
        this.loadingAuthData = true;
        this._cd.markForCheck();
        forkJoin<AuthDriverManifests, Array<AuthConsumer>, Array<AuthSession>>(
            this._authenticationService.getDrivers(),
            this._userService.getConsumers(this.username),
            this._userService.getSessions(this.username)
        )
            .pipe(finalize(() => {
                this.loadingAuthData = false;
                this._cd.markForCheck();
            }))
            .subscribe(res => {
                this.drivers = res[0].manifests.filter(m => m.type !== 'builtin').sort((a, b) => {
                    if (a.type === 'local') {
                        return -1;
                    }
                    if (b.type === 'local') {
                        return 1;
                    }
                    return a.type < b.type ? -1 : 1;
                });
                this.consumers = res[1] ? res[1] : [];

                this.mConsumers = {};
                this.consumers.forEach((c: AuthConsumer) => {
                    this.mConsumers[c.id] = c;
                    if (c.type !== 'builtin') {
                        this.mConsumers[c.type] = c;
                    }
                });

                this.myConsumers = res[1].filter((c: AuthConsumer) => c.type === 'builtin').map((c: AuthConsumer) => {
                    c.parent = this.mConsumers[c.parent_id];
                    return c;
                });

                this.sessions = res[2].map((s: AuthSession) => {
                    s.consumer = this.mConsumers[s.consumer_id];
                    return s;
                });
            });
    }

    modalCreateClose(eventType: CloseEventType) {
        if (eventType === CloseEventType.CREATED) {
            this.getAuthData();
        }
    }

    modalDetailsClose(event: CloseEvent) {
        this.selectedConsumer = null;
        this._cd.markForCheck();

        if (event.type === DetailsCloseEventType.CHILD_DETAILS) {
            this.clickConsumerDetails(event.payload);
            return;
        }

        if (event.type === DetailsCloseEventType.DELETE_OR_DETACH) {
            let currentSession = this.sessions.find(s => s.current);
            if (currentSession && currentSession.consumer_id === event.payload) {
                this._router.navigate(['/auth/signin']);
            } else {
                this.getAuthData();
            }
            return;
        }

        if (event.type === DetailsCloseEventType.REGEN) {
            this.getAuthData();
            return;
        }
    }
}
