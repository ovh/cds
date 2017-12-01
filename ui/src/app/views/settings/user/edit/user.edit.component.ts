import {Component, OnInit} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {User} from '../../../../model/user.model';
import {Group} from '../../../../model/group.model';
import {UserService} from '../../../../service/user/user.service';
import {ToastService} from '../../../../shared/toast/ToastService';
import {TranslateService} from 'ng2-translate';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';

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
    private userPatternError = false;

    constructor(private _userService: UserService,
                private _toast: ToastService, private _translate: TranslateService,
                private _route: ActivatedRoute, private _router: Router,
                private _authentificationStore: AuthentificationStore) {
        this.currentUser = this._authentificationStore.getUser();
    }

    ngOnInit() {
        this._route.params.subscribe(params => {
            this.username = params['username'];
            this._userService.getUser(this.username).subscribe( u => {
                this.user = u;
                this.username = this.user.username;
                this.groups = [];

                this._userService.getGroups(this.user.username).subscribe( g => {
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
            });
        });
    }

    clickDeleteButton(): void {
      this.deleteLoading = true;
      this._userService.deleteUser(this.user.username).subscribe( wm => {
          this.deleteLoading = false;
          this._toast.success('', this._translate.instant('user_deleted'));
          this._router.navigate(['../'], { relativeTo: this._route });
      }, () => {
          this.loading = false;
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
        this._userService.updateUser(this.username, this.user).subscribe( wm => {
            this.loading = false;
            this._toast.success('', this._translate.instant('user_saved'));
            this._router.navigate(['/settings', 'user', this.user.username], { relativeTo: this._route });
        }, () => {
            this.loading = false;
        });
      }

    }
}
