import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { AuthenticationState } from 'app/store/authentication.state';
import { Group } from '../../../../model/group.model';
import { User } from '../../../../model/user.model';
import { UserService } from '../../../../service/user/user.service';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { ToastService } from '../../../../shared/toast/ToastService';
@Component({
    selector: 'app-user-edit',
    templateUrl: './user.edit.html',
    styleUrls: ['./user.edit.scss']
})
export class UserEditComponent implements OnInit {
    loading = false;
    deleteLoading = false;
    user: User;
    currentUser: User;
    groups: Array<Group>;
    groupsAdmin: Array<Group>;
    private username: string;
    private usernamePattern: RegExp = new RegExp('^[a-zA-Z0-9._-]{1,}$');
    userPatternError = false;
    path: Array<PathItem>;

    constructor(
        private _userService: UserService,
        private _toast: ToastService, private _translate: TranslateService,
        private _route: ActivatedRoute, private _router: Router,
        private _store: Store
    ) {
        this.currentUser = this._store.selectSnapshot(AuthenticationState.user);
    }

    ngOnInit() {
        this._route.params.subscribe(params => {
            this.username = params['username'];

            this._userService.getUser(this.username).subscribe(u => {
                this.user = u;
                this.username = this.user.username;
                this.groups = [];

                this._userService.getGroups(this.user.username).subscribe(g => {
                    this.groupsAdmin = g.groups_admin;
                    for (let i = 0; i < g.groups.length; i++) {
                        let userAdminOnGroup = false;
                        for (let j = 0; j < this.groupsAdmin.length; j++) {
                            if (this.groupsAdmin[j].name === g.groups[i].name) {
                                userAdminOnGroup = true;
                                break;
                            }
                        }
                        if (!userAdminOnGroup) {
                            this.groups.push(g.groups[i]);
                        }
                    }
                });

                this.updatePath();
            });
        });
    }

    clickDeleteButton(): void {
        this.deleteLoading = true;
        this._userService.deleteUser(this.user.username).subscribe(wm => {
            this.deleteLoading = false;
            this._toast.success('', this._translate.instant('user_deleted'));
            this._router.navigate(['../'], { relativeTo: this._route });
        }, () => {
            this.deleteLoading = false;
        });
    }

    clickSaveButton(): void {
        if (!this.user.username) {
            return;
        }

        if (!this.usernamePattern.test(this.user.username)) {
            this.userPatternError = true;
            return;
        }

        this.loading = true;
        if (this.user.id > 0) {
            this._userService.updateUser(this.username, this.user).subscribe(wm => {
                this.loading = false;
                this._toast.success('', this._translate.instant('user_saved'));
                this._router.navigate(['/settings', 'user', this.user.username], { relativeTo: this._route });
            }, () => {
                this.loading = false;
            });
        }
    }

    updatePath() {
        this.path = [<PathItem>{
            translate: 'common_settings'
        }, <PathItem>{
            translate: 'user_list_title',
            routerLink: ['/', 'settings', 'user']
        }];

        if (this.user && this.user.id) {
            this.path.push(<PathItem>{
                text: this.user.username,
                routerLink: ['/', 'settings', 'user', this.user.username]
            });
        }
    }
}
