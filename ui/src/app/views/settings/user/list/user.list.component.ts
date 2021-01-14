import { ChangeDetectionStrategy, ChangeDetectorRef, Component } from '@angular/core';
import { AuthentifiedUser } from 'app/model/user.model';
import { UserService } from 'app/service/user/user.service';
import { PathItem } from 'app/shared/breadcrumb/breadcrumb.component';
import { Column, ColumnType } from 'app/shared/table/data-table.component';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-user-list',
    templateUrl: './user.list.html',
    styleUrls: ['./user.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class UserListComponent {
    loading: boolean;
    columns: Array<Column<AuthentifiedUser>>;
    users: Array<AuthentifiedUser>;
    path: Array<PathItem>;

    constructor(
        private _userService: UserService,
        private _cd: ChangeDetectorRef
    ) {
        this.columns = [
            <Column<AuthentifiedUser>>{
                type: ColumnType.ICON,
                class: 'one',
                selector: (u: AuthentifiedUser) => u.ring === 'ADMIN' ? ['user', 'outline', 'icon'] : ['user', 'icon']
            },
            <Column<AuthentifiedUser>>{
                type: ColumnType.ROUTER_LINK,
                class: 'six',
                name: 'user_label_username',
                selector: (u: AuthentifiedUser) => ({
                    link: `/settings/user/${u.username}`,
                    value: u.username
                })
            },
            <Column<AuthentifiedUser>>{
                type: ColumnType.TEXT,
                class: 'six',
                name: 'user_label_fullname',
                selector: (u: AuthentifiedUser) => u.username
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
            .subscribe(us => {
                this.users = us;
            });
    }

    filter(f: string) {
        const lowerFilter = f.toLowerCase();
        return (u: AuthentifiedUser) => u.username.toLowerCase().indexOf(lowerFilter) !== -1 ||
            u.fullname.toLowerCase().indexOf(lowerFilter) !== -1
    }
}
