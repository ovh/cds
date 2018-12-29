import { Component, Input } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { User } from '../../../../model/user.model';
import { AuthentificationStore } from '../../../../service/auth/authentification.store';
import { UserService } from '../../../../service/user/user.service';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { Table } from '../../../../shared/table/table';
import { ToastService } from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-user-list',
    templateUrl: './user.list.html',
    styleUrls: ['./user.list.scss']
})
export class UserListComponent extends Table<User> {
    currentUser: User;
    filter: string;
    users: Array<User>;
    path: Array<PathItem>;

    @Input('maxPerPage')
    set maxPerPage(data: number) {
        this.nbElementsByPage = data;
    };

    constructor(
        private _userService: UserService,
        private _toast: ToastService,
        private _authentificationStore: AuthentificationStore,
        private _translate: TranslateService
    ) {
        super();

        this.currentUser = this._authentificationStore.getUser();

        // list only for admin
        if (this.currentUser.admin) {
            this._userService.getUsers().subscribe(users => {
                this.users = users;
            });
        } else {
            this._toast.error('', this._translate.instant('access_refused'));
        }

        this.path = [<PathItem>{
            translate: 'common_settings'
        }, <PathItem>{
            translate: 'user_list_title',
            routerLink: ['/', 'settings', 'user']
        }];
    }

    getData(): Array<User> {
        if (!this.filter) {
            return this.users;
        }
        return this.users.filter(v => v.username.indexOf(this.filter) !== -1);
    }
}
