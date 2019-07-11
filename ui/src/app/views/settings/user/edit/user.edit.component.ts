import { Component, OnInit } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { AuthConsumer, AuthDriverManifest, AuthSession } from 'app/model/authentication.model';
import { AuthenticationService } from 'app/service/services.module';
import { Item } from 'app/shared/menu/menu.component';
import { Column, ColumnType, Filter } from 'app/shared/table/data-table.component';
import { AuthenticationState } from 'app/store/authentication.state';
import { forkJoin } from 'rxjs/internal/observable/forkJoin';
import { finalize } from 'rxjs/operators/finalize';
import { Group } from '../../../../model/group.model';
import { User, UserContact } from '../../../../model/user.model';
import { UserService } from '../../../../service/user/user.service';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';

@Component({
    selector: 'app-user-edit',
    templateUrl: './user.edit.html',
    styleUrls: ['./user.edit.scss']
})
export class UserEditComponent implements OnInit {
    loading = false;
    deleteLoading = false;
    groupsAdmin: Array<Group>;
    // private usernamePattern: RegExp = new RegExp('^[a-zA-Z0-9._-]{1,}$');
    userPatternError = false;

    username: string;
    currentUser: User;
    path: Array<PathItem>;
    items: Array<Item>;
    selectedItem: Item;
    loadingUser: boolean;
    user: User;
    columnsGroups: Array<Column<Group>>;
    loadingGroups = false;
    groups: Array<Group>;
    columnsContacts: Array<Column<UserContact>>;
    loadingContacts = false;
    contacts: Array<UserContact>;
    loadingAuthData: boolean;
    drivers: Array<AuthDriverManifest>;
    consumers: Array<AuthConsumer>;
    myConsumers: Array<AuthConsumer>;
    columnsConsumers: Array<Column<AuthConsumer>>;
    filterConsumers: Filter<AuthConsumer>;
    columnsSessions: Array<Column<AuthSession>>;
    filterSessions: Filter<AuthSession>;
    sessions: Array<AuthSession>;

    constructor(
        private _authenticationService: AuthenticationService,
        private _userService: UserService,
        private _translate: TranslateService,
        private _route: ActivatedRoute,
        private _store: Store
    ) {
        this.currentUser = this._store.selectSnapshot(AuthenticationState.user);

        this.items = [<Item>{
            translate: 'user_profile_btn',
            key: 'profile',
            default: true
        }, <Item>{
            translate: 'user_groups_btn',
            key: 'groups'
        }, <Item>{
            translate: 'user_contacts_btn',
            key: 'contacts'
        }, <Item>{
            translate: 'user_authentication_btn',
            key: 'authentication'
        }];

        this.columnsGroups = [
            <Column<Group>>{
                name: 'common_name',
                selector: (g: Group) => g.name
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
            return (c: AuthConsumer) => {
                return c.name.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    c.description.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    c.scopes.join(' ').toLowerCase().indexOf(lowerFilter) !== -1;
            }
        };

        this.columnsConsumers = [
            <Column<AuthConsumer>>{
                name: 'common_name',
                selector: (c: AuthConsumer) => c.name
            },
            <Column<AuthConsumer>>{
                name: 'common_description',
                selector: (c: AuthConsumer) => c.description
            },
            <Column<AuthConsumer>>{
                name: 'user_auth_scopes',
                selector: (c: AuthConsumer) => c.scopes.join(', ')
            },
            <Column<AuthConsumer>>{
                name: 'user_auth_groups_count',
                selector: (c: AuthConsumer) => c.group_ids ? c.group_ids.length : '*'
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
                name: 'user_auth_consumer',
                selector: (s: AuthSession) => s.consumer.name + ' (' + s.consumer_id + ')'
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
                selector: (s: AuthSession) => {
                    return {
                        title: 'user_auth_revoke_btn',
                        click: () => { this.clickSessionRevoke(s) }
                    };
                }
            }
        ];
    }

    ngOnInit() {
        this._route.params.subscribe(params => {
            this.username = params['username'];
            this.getUser();
        });
    }

    clickConsumerDetails(c: AuthConsumer): void {

    }

    clickSessionRevoke(s: AuthSession): void {

    }

    clickDeleteButton(): void {
        // this.deleteLoading = true;
        // this._userService.deleteUser(this.currentUser.username).subscribe(wm => {
        //     this.deleteLoading = false;
        //     this._toast.success('', this._translate.instant('user_deleted'));
        //     this._router.navigate(['../'], { relativeTo: this._route });
        // }, () => {
        //     this.deleteLoading = false;
        // });
    }

    clickSaveButton(): void {
        // if (!this.user.username) {
        //     return;
        // }
        //
        // if (!this.usernamePattern.test(this.user.username)) {
        //     this.userPatternError = true;
        //     return;
        // }

        // this.loading = true;
        // if (this.user.id > 0) {
        //    this._userService.updateUser(this.username, this.user).subscribe(wm => {
        //        this.loading = false;
        //        this._toast.success('', this._translate.instant('user_saved'));
        //        this._router.navigate(['/settings', 'user', this.user.username], { relativeTo: this._route });
        //    }, () => {
        //        this.loading = false;
        //    });
        // }
    }

    updatePath() {
        this.path = [<PathItem>{
            translate: 'common_settings'
        }, <PathItem>{
            translate: 'user_list_title',
            routerLink: ['/', 'settings', 'user']
        }, <PathItem>{
            text: this.user.username,
            routerLink: ['/', 'settings', 'user', this.currentUser.username]
        }];
    }

    selectItem(item: Item): void {
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
    }

    getUser(): void {
        this.loadingUser = true;
        this._userService.getUser(this.username)
            .pipe(finalize(() => this.loadingUser = false))
            .subscribe(u => {
                this.user = u;
                this.updatePath();
            });
    }

    getGroups(): void {
        this.loadingGroups = true;
        this._userService.getGroups(this.currentUser.username)
            .pipe(finalize(() => this.loadingGroups = false))
            .subscribe((gs) => {
                this.groups = gs;
            });
    }

    getContacts(): void {
        this.loadingContacts = true;
        this._userService.getContacts(this.currentUser.username)
            .pipe(finalize(() => this.loadingContacts = false))
            .subscribe((cs) => {
                this.contacts = cs;
            });
    }

    getAuthData(): void {
        this.loadingAuthData = true;
        forkJoin<Array<AuthDriverManifest>, Array<AuthConsumer>, Array<AuthSession>>(
            this._authenticationService.getDrivers(),
            this._userService.getConsumers(this.currentUser.username),
            this._userService.getSessions(this.currentUser.username)
        )
            .pipe(finalize(() => this.loadingAuthData = false))
            .subscribe(res => {
                this.drivers = res[0].filter(m => m.type !== 'builtin');
                this.consumers = res[1];

                let mConsumers = {};
                this.consumers.forEach((c: AuthConsumer) => { mConsumers[c.id] = c });

                this.myConsumers = res[1].filter((c: AuthConsumer) => c.type === 'builtin').map((c: AuthConsumer) => {
                    c.parent = mConsumers[c.parent_id];
                    return c;
                });

                this.sessions = res[2].map((s: AuthSession) => {
                    s.consumer = mConsumers[s.consumer_id];
                    return s;
                });
            });
    }
}
