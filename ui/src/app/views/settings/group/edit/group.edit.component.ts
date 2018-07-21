import {Component, OnInit} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {TranslateService} from '@ngx-translate/core';
import {finalize, first} from 'rxjs/operators';
import {Group} from '../../../../model/group.model';
import {Token, TokenEvent} from '../../../../model/token.model';
import {User} from '../../../../model/user.model';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';
import {GroupService} from '../../../../service/group/group.service';
import {UserService} from '../../../../service/user/user.service';
import {ToastService} from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-group-edit',
    templateUrl: './group.edit.html',
    styleUrls: ['./group.edit.scss']
})
export class GroupEditComponent implements OnInit {
    loading = false;
    deleteLoading = false;
    group: Group;
    currentUser: User;
    currentUserIsAdminOnGroup: boolean;
    addUserUsername: string;
    users: Array<User>;
    members: Array<User>;
    tokenGenerated: Token;

    private groupname: string;
    private groupnamePattern: RegExp = new RegExp('^[a-zA-Z0-9._-]{1,}$');
    groupPatternError = false;

    constructor(private _userService: UserService, private _groupService: GroupService,
                private _toast: ToastService, private _translate: TranslateService,
                private _route: ActivatedRoute, private _router: Router,
                private _authentificationStore: AuthentificationStore) {
        this.currentUser = this._authentificationStore.getUser();
        this._userService.getUsers().subscribe( users => {
            this.users = users;
        });
    }

    ngOnInit() {
        this._route.params.subscribe(params => {
            if (params['groupname'] !== 'add') {
                this.reloadData(params['groupname']);
            } else {
                this.group = new Group();
            }
        });
    }

    reloadData(groupname: string): void {
      this._groupService.getGroupByName(groupname).subscribe( wm => {
          this.group = wm;
          this.groupname = wm.name;
          this.members = new Array<User>();
          if (wm.admins && wm.admins.length > 0) {
            for (let i = 0; i < wm.admins.length; i++) {
                let u = wm.admins[i];
                u.admin = true;
                this.members.push(u);
                if (this.currentUser.username === u.username) {
                  this.currentUserIsAdminOnGroup = true;
                }
            }
          }
          if (wm.users && wm.users.length > 0) {
            for (let i = 0; i < wm.users.length; i++) {
                let u = wm.users[i];
                u.admin = false;
                this.members.push(u);
            }
          }
      });
    }

    clickDeleteButton(): void {
      this.deleteLoading = true;
      this._groupService.deleteGroup(this.group.name)
        .pipe(
          finalize(() => this.deleteLoading = false)
        )
        .subscribe( wm => {
            this._toast.success('', this._translate.instant('group_deleted'));
            this._router.navigate(['../'], { relativeTo: this._route });
        });
    }

    clickSaveButton(): void {
      if (!this.group.name) {
          return;
      }

      if (!this.groupnamePattern.test(this.group.name)) {
          this.groupPatternError = true;
          return;
      }

      this.loading = true;
      if (this.group.id > 0) {
        this._groupService.updateGroup(this.groupname, this.group)
            .pipe(
              finalize(() => this.loading = false)
            )
            .subscribe( wm => {
                this._toast.success('', this._translate.instant('group_saved'));
                this._router.navigate(['settings', 'group', this.group.name]);
            });
      } else {
        this._groupService.createGroup(this.group)
            .pipe(
              finalize(() => this.loading = false)
            )
            .subscribe( wm => {
                this._toast.success('', this._translate.instant('group_saved'));
                this._router.navigate(['settings', 'group', this.group.name]);
            });
      }
    }

    clickAddAdminButton(username: string): void {
      this.loading = true;
      this._groupService.addUserAdmin(this.group.name, username)
          .pipe(
            finalize(() => this.loading = false)
          )
          .subscribe( wm => {
              this._toast.success('', this._translate.instant('group_add_admin_saved'));
              this.reloadData(this.group.name);
          });
    }

    clickRemoveAdminButton(username: string): void {
      this.loading = true;
      this._groupService.removeUserAdmin(this.group.name, username)
          .pipe(
              finalize(() => this.loading = false)
          )
          .subscribe( wm => {
              this._toast.success('', this._translate.instant('group_remove_admin_saved'));
              this.reloadData(this.group.name);
          });
    }

    clickRemoveUserButton(username: string): void {
      this.loading = true;
      this._groupService.removeUser(this.group.name, username)
          .pipe(
            finalize(() => this.loading = false)
          )
          .subscribe( wm => {
              this._toast.success('', this._translate.instant('group_remove_user_saved'));
              this.reloadData(this.group.name);
          });
    }

    clickAddUserButton(): void {
      this.loading = true;
      this._groupService.addUser(this.group.name, this.addUserUsername).subscribe(() => {
          this.loading = false;
          this._toast.success('', this._translate.instant('group_add_user_saved'));
          this.reloadData(this.group.name);
      }, () => {
          this.loading = false;
      });
    }

    tokenEvent(event: TokenEvent): void {
        if (!event) {
            return;
        }
        switch (event.type) {
            case 'delete':
                this._groupService.removeToken(this.groupname, event.token.id)
                    .pipe(
                        first(),
                        finalize(() => event.token.updating = false)
                    )
                    .subscribe(() => {
                        this.group.tokens = this.group.tokens.filter((token) => token.id !== event.token.id);
                        this._toast.success('', this._translate.instant('token_deleted'));
                    });
                    break;
            case 'add':
                this._groupService.addToken(this.groupname, event.token.expirationString, event.token.description)
                    .pipe(
                        first(),
                        finalize(() => {
                            event.token.expirationString = null;
                            event.token.description = null;
                            event.token.updating = false;
                        })
                    )
                    .subscribe((token) => {
                        if (!Array.isArray(this.group.tokens)) {
                            this.group.tokens = [token];
                        } else {
                            this.group.tokens.push(token);
                        }
                        this._toast.success('', this._translate.instant('token_added'));
                        this.tokenGenerated = token;
                    });
                    break;
        }
    }
}
