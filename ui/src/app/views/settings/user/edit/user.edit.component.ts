import {Component, OnInit} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {finalize, first} from 'rxjs/operators';
import {User} from '../../../../model/user.model';
import {Token} from '../../../../model/token.model';
import {Group} from '../../../../model/group.model';
import {UserService} from '../../../../service/user/user.service';
import {GroupService} from '../../../../service/group/group.service';
import {ToastService} from '../../../../shared/toast/ToastService';
import {TranslateService} from '@ngx-translate/core';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';

@Component({
    selector: 'app-user-edit',
    templateUrl: './user.edit.html',
    styleUrls: ['./user.edit.scss']
})
export class UserEditComponent implements OnInit {
    loading = false;
    deleteLoading = false;
    tokensLoading = true;
    user: User;
    currentUser: User;
    groups: Array<Group>;
    groupsAdmin: Array<Group>;
    tokens: Array<Token>;

    private username: string;
    private usernamePattern: RegExp = new RegExp('^[a-zA-Z0-9._-]{1,}$');
    private userPatternError = false;

    constructor(private _userService: UserService,
                private _toast: ToastService, private _translate: TranslateService,
                private _route: ActivatedRoute, private _router: Router,
                private _authentificationStore: AuthentificationStore,
                private _groupService: GroupService) {
        this.currentUser = this._authentificationStore.getUser();
        this.tokensLoading = true;
        this._userService.getTokens()
            .pipe(finalize(() => this.tokensLoading = false))
            .subscribe((tokens) => this.tokens = tokens);
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
        this._userService.updateUser(this.username, this.user).subscribe( wm => {
            this.loading = false;
            this._toast.success('', this._translate.instant('user_saved'));
            this._router.navigate(['/settings', 'user', this.user.username], { relativeTo: this._route });
        }, () => {
            this.loading = false;
        });
      }

    }

    tokenEvent(event: TokenEvent): void {
        if (!event) {
            return;
        }
        switch (event.type) {
            case 'delete':
                this._groupService.removeToken(event.token.group_name, event.token.id)
                    .pipe(
                        first(),
                        finalize(() => event.token.updating = false)
                    )
                    .subscribe(() => {
                        this.tokens = this.tokens.filter((token) => token.id !== event.token.id);
                        this._toast.success('', this._translate.instant('token_deleted'));
                    });
                    break;
            case 'add':
                this._groupService.addToken(event.token.group_name, event.token.expirationString, event.token.description)
                    .pipe(
                        first(),
                        finalize(() => {
                            event.token.expirationString = null;
                            event.token.description = null;
                            event.token.updating = false;
                        })
                    )
                    .subscribe((token) => {
                        if (!Array.isArray(this.tokens)) {
                            this.tokens = [token];
                        } else {
                            this.tokens.push(token);
                        }
                        this._toast.success('', this._translate.instant('token_added'));
                    });
                    break;
        }
    }
}
