import {Component, Input} from '@angular/core';
import {TranslateService} from 'ng2-translate';
import {User} from '../../../../model/user.model';
import {Table} from '../../../../shared/table/table';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';
import {UserService} from '../../../../service/user/user.service';
import {ToastService} from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-user-list',
    templateUrl: './user.list.html',
    styleUrls: ['./user.list.scss']
})
export class UserListComponent extends Table {
    public currentUser: User;

    @Input('maxPerPage')
    set maxPerPage(data: number) {
        this.nbElementsByPage = data;
    };

    public ready = true;
    filter: string;
    users: Array<User>;

    constructor(private _userService: UserService,
                private _toast: ToastService,
                private _authentificationStore: AuthentificationStore,
                private _translate: TranslateService) {
        super();
        this.currentUser = this._authentificationStore.getUser();
        // list only for admin
        if (this.currentUser.admin) {
          this._userService.getUsers().subscribe( users => {
              this.users = users;
              this.ready = true;
          });
        } else {
          this._toast.error('', this._translate.instant('access_refused'));
          this.ready = true;
        }
    }

    getData(): any[] {
        if (!this.filter || this.filter === '') {
            return this.users;
        }
        return this.users.filter(v => v.username.indexOf(this.filter) !== -1);
    }
}
