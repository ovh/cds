import { ChangeDetectionStrategy, ChangeDetectorRef, Component } from '@angular/core';
import { finalize } from 'rxjs/operators/finalize';
import { AuthentifiedUser, User } from '../../../../model/user.model';
import { UserService } from '../../../../service/user/user.service';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { Column, ColumnType } from '../../../../shared/table/data-table.component';

@Component({
    selector: 'app-user-list',
    templateUrl: './user.list.html',
    styleUrls: ['./user.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class UserListComponent {
    loading: boolean;
    columns: Array<Column<User>>;
    users: Array<AuthentifiedUser>;
    path: Array<PathItem>;

    constructor(
        private _userService: UserService,
        private _cd: ChangeDetectorRef
    ) {
        this.columns = [
            <Column<User>>{
                type: ColumnType.ICON,
                class: 'one',
                selector: (u: User) => { return u.admin ? ['user', 'outline', 'icon'] : ['user', 'icon']; }
            },
            <Column<User>>{
                type: ColumnType.ROUTER_LINK,
                class: 'five',
                name: 'user_label_username',
                selector: (u: User) => {
                    return {
                        link: '/settings/user/' + u.username,
                        value: u.username
                    };
                }
            },
            <Column<User>>{
                type: ColumnType.TEXT,
                class: 'five',
                name: 'user_label_fullname',
                selector: (u: User) => { return u.username; }
            },
            <Column<User>>{
                type: ColumnType.TEXT,
                class: 'five',
                name: 'user_label_email',
                selector: (u: User) => { return u.email; }
            }
        ];
        this.getUsers();

        this.path = [<PathItem>{
            translate: 'common_settings'
        }, <PathItem>{
            translate: 'user_list_title',
            routerLink: ['/', 'settings', 'user']
        }];
    }

    getUsers(): void {
        this.loading = true;
        this._userService.getUsers()
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(us => { this.users = us; });
    }

    filter(f: string) {
        const lowerFilter = f.toLowerCase();
        return (u: User) => {
            return u.username.toLowerCase().indexOf(lowerFilter) !== -1 ||
                u.email.toLowerCase().indexOf(lowerFilter) !== -1 ||
                u.fullname.toLowerCase().indexOf(lowerFilter) !== -1;
        }
    }
}
