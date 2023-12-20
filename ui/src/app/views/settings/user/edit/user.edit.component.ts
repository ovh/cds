import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnInit, TemplateRef, ViewChild } from '@angular/core';
import { NgForm } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { AuthConsumer, AuthDriverManifest, AuthDriverManifests, AuthSession } from 'app/model/authentication.model';
import { Group } from 'app/model/group.model';
import {AuthentifiedUser, AuthSummary, UserContact, UserLink} from 'app/model/user.model';
import { AuthenticationService } from 'app/service/authentication/authentication.service';
import { UserService } from 'app/service/user/user.service';
import { PathItem } from 'app/shared/breadcrumb/breadcrumb.component';
import { Column, ColumnType, Filter } from 'app/shared/table/data-table.component';
import { ToastService } from 'app/shared/toast/ToastService';
import { AuthenticationState } from 'app/store/authentication.state';
import moment from 'moment';
import { forkJoin } from 'rxjs';
import { finalize, first } from 'rxjs/operators';
import { ConsumerCreateModalComponent } from '../consumer-create-modal/consumer-create-modal.component';
import {
    CloseEvent,
    CloseEventType as DetailsCloseEventType,
    ConsumerDetailsModalComponent
} from '../consumer-details-modal/consumer-details-modal.component';
import { NzModalService } from 'ng-zorro-antd/modal';
import {HttpClient} from "@angular/common/http";
import {LinkService} from "../../../../service/link/link.service";

const usernamePattern = new RegExp('^[a-zA-Z0-9._-]{1,}$');

@Component({
    selector: 'app-user-edit',
    templateUrl: './user.edit.html',
    styleUrls: ['./user.edit.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class UserEditComponent implements OnInit {

    @ViewChild('ldapSigninForm')
    ldapSigninForm: NgForm;

    @ViewChild('modalHeaderTmpl')
    modalTitleTmpl: TemplateRef<any>

    loading = false;
    deleteLoading = false;
    userPatternError = false;
    username: string;
    currentAuthSummary: AuthSummary;
    editable: boolean;
    path: Array<PathItem>;
    menuItems: Map<string,string>;
    selectedItem: string;
    loadingUser: boolean;
    user: AuthentifiedUser;
    userLinks: Map<string, UserLink>;
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

    linkDriver: string[] = [];

    constructor(
        private _authenticationService: AuthenticationService,
        private _userService: UserService,
        private _translate: TranslateService,
        private _route: ActivatedRoute,
        private _router: Router,
        private _store: Store,
        private _toast: ToastService,
        private _cd: ChangeDetectorRef,
        private _modalService: NzModalService,
        private _http: HttpClient,
        private _linkService: LinkService
    ) {
        this.currentAuthSummary = this._store.selectSnapshot(AuthenticationState.summary);

        this._linkService.getDrivers()
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe((data) => {
                this.linkDriver = data;
            });

        this.menuItems = new Map<string, string>();
        this.selectedItem = "profile";

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
            }
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
                        labels.push({ color: 'success', title: 'user_contact_primary' });
                    }
                    if (!c.verified) {
                        labels.push({ color: 'error', title: 'user_contact_not_verified' });
                    }

                    return {
                        value: c.value,
                        labels
                    };
                }
            }
        ];

        this.filterConsumers = f => {
            const lowerFilter = f.toLowerCase();
            return (c: AuthConsumer) => c.name.toLowerCase().indexOf(lowerFilter) !== -1 ||
                c.description.toLowerCase().indexOf(lowerFilter) !== -1 ||
                c.id.toLowerCase().indexOf(lowerFilter) !== -1 ||
                c.auth_consumer_user.scope_details.map(s => s.scope).join(' ').toLowerCase().indexOf(lowerFilter) !== -1 ||
                (c.auth_consumer_user.groups && c.auth_consumer_user.groups.map(g => g.name).join(' ').toLowerCase().indexOf(lowerFilter) !== -1) ||
                (!c.auth_consumer_user.groups && lowerFilter === '*');
        };

        this.columnsConsumers = [
            <Column<AuthConsumer>>{
                type: ColumnType.TEXT_LABELS,
                name: 'Name',
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
                name: 'common_description',
                selector: (c: AuthConsumer) => c.description
            },
            <Column<AuthConsumer>>{
                name: 'user_auth_scopes',
                selector: (c: AuthConsumer) => c.auth_consumer_user.scope_details.map(s => s.scope).join(', ')
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
                            type: 'warning',
                            theme: 'outline',
                            class: ['orange'],
                            title: text
                        });
                    }

                    return {
                        value: c.auth_consumer_user.groups ? c.auth_consumer_user.groups.map((g: Group) => g.name).join(', ') : '*',
                        icons
                    };
                }
            },
            <Column<AuthConsumer>>{
                name: 'End of token validity',
                selector: (c: AuthConsumer) => {
                    if (!c.validity_periods) {
                        return '';
                    }

                    c.validity_periods.sort((x, y) => {
                        let dX = moment(x.issued_at).toDate();
                        let dY = moment(y.issued_at).toDate();
                        return dY.getTime() - dX.getTime();
                    });

                    let period = c.validity_periods[0];
                    if (period.duration === 0) {
                        return '';
                    }

                    let d = moment(period.issued_at).toDate();
                    d.setTime(d.getTime() + (period.duration / 1000000));

                    return moment(d).fromNow();
                }
            },
            <Column<AuthConsumer>>{
                name: 'Last authentication',
                selector: (c: AuthConsumer) => {
                    if (!c.last_authentication) {
                        return 'never';
                    }
                    return moment(c.last_authentication).fromNow();
                }
            },
            <Column<AuthConsumer>>{
                type: ColumnType.BUTTON,
                name: 'Action',
                class: 'rightAlign',
                selector: (c: AuthConsumer) => ({
                    title: 'Details',
                    buttonDanger: false,
                    click: () => {
                        this.clickConsumerDetails(c);
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
                s.expire_at.toLowerCase().indexOf(lowerFilter) !== -1;
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
                            theme: 'outline',
                            type: '',
                            label: `ID: ${s.consumer_id}`,
                            title: `ID: ${s.consumer_id}`,
                        };
                        switch (consumer.type) {
                            case 'builtin':
                                icon.type = 'info-circle';
                                break;
                            case 'local':
                                icon.type = 'lock';
                                break;
                            case 'ldap':
                                icon.type = 'audit';
                                break;
                            case 'corporate-sso':
                                icon.type = 'safety-certificate';
                                break;
                            case 'openid-connect':
                                icon.type = 'safety-certificate';
                                break;
                            default:
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
            },
            <Column<AuthSession>>{
                type: ColumnType.CONFIRM_BUTTON,
                name: 'Action',
                class: 'two right aligned',
                disabled: true,
                selector: (s: AuthSession) => ({
                    buttonType: 'primary',
                    buttonDanger: true,
                    buttonConfirmationMessage: 'Are you sure you want to revoke this session ?',
                    title: 'Revoke',
                    click: () => {
                        this.clickSessionRevoke(s);
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

    linkUser(type: string): void {
        this._linkService.askLink(type, "/settings/user/" + this.user.username)
            .pipe(first())
            .subscribe(redirect  => {
                if (redirect.method.toLowerCase() === ('get')) {
                    window.location.replace(redirect.url);
                }
        })
    }

    clickConsumerDetails(selected: AuthConsumer): void {
        this.selectedConsumer = selected;

        // calculate children for selected consumer
        this.selectedConsumer.children = this.consumers.filter(c => c.parent_id === this.selectedConsumer.id);
        this.selectedConsumer.sessions = this.sessions.filter(s => s.consumer_id === this.selectedConsumer.id);

        this._cd.detectChanges(); // manually ask for detect changes to allow modal data to be set before opening
        let modal = this._modalService.create({
            nzTitle: this.modalTitleTmpl,
            nzWidth: '900px',
            nzContent: ConsumerDetailsModalComponent,
            nzData: {
                consumer: this.selectedConsumer,
                user: this.user,
            },
            nzFooter: null,
        });


        modal.afterClose.subscribe(t => {
            this.modalDetailsClose(t);
        })
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
        this.showLDAPSigninForm = true;
        this._cd.detectChanges();
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
        this._modalService.create({
            nzTitle: 'Create a new consumer',
            nzWidth: '900px',
            nzContent: ConsumerCreateModalComponent,
            nzData: {
                user: this.user
            },
            nzOnOk: () => {
                this.getAuthData();
            }
        });
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

    selectMenuItem(item: string): void {
        switch (item) {
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
        this._router.navigate([], {
            relativeTo: this._route,
            queryParams: { item: item },
            queryParamsHandling: 'merge'
        });
        this._cd.markForCheck();
    }

    getUser(): void {
        this.loadingUser = true;
        this._cd.markForCheck();

        forkJoin([
            this._userService.get(this.username),
            this._userService.getLinks(this.username)
        ]).pipe(finalize(() => {
            this.loadingUser = false;
            this._cd.markForCheck();
        })).subscribe(result => {
            this.user = result[0];
            if (result[1]) {
                this.userLinks = new Map<string, UserLink>();
                result[1].forEach(l => {
                    this.userLinks.set(l.type, l);
                })
            }
            this.setDataFromUser();
            this.updatePath();
        });
    }

    setDataFromUser(): void {
        this.editable = this.user.id === this.currentAuthSummary.user.id || this.currentAuthSummary.isAdmin();
        this.menuItems = new Map<string, string>();
        this.menuItems.set("profile", "Profile");
        this.menuItems.set("groups", "Groups");
        if (this.linkDriver.length > 0) {
            this.menuItems.set("links", "Links");
        }
        if (this.user.id === this.currentAuthSummary.user.id || this.currentAuthSummary.isMaintainer()) {
            this.menuItems.set("contacts", "Contacts");
            this.menuItems.set("authentication", "Authentication");
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

    modalDetailsClose(event: CloseEvent) {
        this.selectedConsumer = null;
        this._cd.markForCheck();

        if (event?.type === DetailsCloseEventType.CHILD_DETAILS) {
            this.clickConsumerDetails(event.payload);
            return;
        }

        if (event?.type === DetailsCloseEventType.DELETE_OR_DETACH) {
            let currentSession = this.sessions.find(s => s.current);
            if (currentSession && currentSession.consumer_id === event.payload) {
                this._router.navigate(['/auth/signin']);
            } else {
                this.getAuthData();
            }
            return;
        }

        if (event?.type === DetailsCloseEventType.REGEN) {
            this.getAuthData();
            return;
        }
    }
}
