import { Component, OnInit } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Item } from 'app/shared/menu/menu.component';
import { Column, ColumnType } from 'app/shared/table/data-table.component';
import { AuthenticationState } from 'app/store/authentication.state';
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

    constructor(
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
    }

    ngOnInit() {
        this._route.params.subscribe(params => {
            this.username = params['username'];
            this.getUser();
        });
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
}
