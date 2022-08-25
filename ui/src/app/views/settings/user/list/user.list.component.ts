import { ChangeDetectionStrategy, ChangeDetectorRef, Component } from '@angular/core';
import { AuthentifiedUser } from 'app/model/user.model';
import { UserService } from 'app/service/user/user.service';
import { PathItem } from 'app/shared/breadcrumb/breadcrumb.component';
import { Column, ColumnType, DataTableComponent } from 'app/shared/table/data-table.component';
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
                selector: (u: AuthentifiedUser) => ({
                    title: u.ring,
                    iconTheme: 'outline',
                    iconType: u.ring === 'ADMIN' ? 'crown' : 'user'
                })
            },
            <Column<AuthentifiedUser>>{
                type: ColumnType.ROUTER_LINK,
                name: 'user_label_username',
                selector: (u: AuthentifiedUser) => ({
                    link: `/settings/user/${u.username}`,
                    value: u.username
                })
            },
            <Column<AuthentifiedUser>>{
                type: ColumnType.TEXT,
                name: 'user_label_fullname',
                selector: (u: AuthentifiedUser) => u.username
            },
            <Column<AuthentifiedUser>>{
                type: ColumnType.TEXT,
                class: 'two',
                name: 'Organization',
                disabled: true,
                selector: (u: AuthentifiedUser) => u.organization
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
                const userWithOrg = this.users.map(u => !!u.organization).reduce((p, c) => p || c);
                this.columns[3].disabled = !userWithOrg;
            });
    }

    filter(rawSearch: string) {
        return DataTableComponent.filterArgsFunc(rawSearch, (search: string, u: AuthentifiedUser) =>
            u.username.toLowerCase().indexOf(search) !== -1 || u.fullname.toLowerCase().indexOf(search) !== -1
        );
    }
}
