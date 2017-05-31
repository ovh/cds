import {Component, Input, OnInit, ViewChild} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {Group} from '../../../../model/group.model';
import {GroupService} from '../../../../service/group/group.service';
import {User} from '../../../../model/user.model';
import {UserService} from '../../../../service/user/user.service';
import {Subscription} from 'rxjs/Subscription';
import {ToastService} from '../../../../shared/toast/ToastService';
import {TranslateService} from 'ng2-translate';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';

@Component({
    selector: 'app-group-edit',
    templateUrl: './group.edit.html',
    styleUrls: ['./group.edit.scss']
})
export class GroupEditComponent implements OnInit {
    public ready = true;
    public loadingSave = false;
    public deleteLoading = false;
    public group: Group;
    public currentUser: User;
    public currentUserIsAdminOnGroup: boolean;
    private groupname: string;

    private groupnamePattern: RegExp = new RegExp('^[a-zA-Z0-9._-]{1,}$');
    private groupPatternError = false;

    public addUserUsername: string;
    public users: Array<User>;
    public members: Array<User>;

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
                this.ready = true;
            }
        });
    }

    reloadData(groupname: string): void {
      this._groupService.getGroupByName(groupname).subscribe( wm => {
          this.group = wm;
          this.groupname = wm.name;
          this.ready = true;
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
      this._groupService.deleteGroup(this.group.name).subscribe( wm => {
          this.deleteLoading = false;
          this._toast.success('', this._translate.instant('group_deleted'));
          this._router.navigate(['../'], { relativeTo: this._route });
      }, () => {
          this.loadingSave = false;
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

      this.loadingSave = true;
      if (this.group.id > 0) {
        this._groupService.updateGroup(this.groupname, this.group).subscribe( wm => {
            this.loadingSave = false;
            this._toast.success('', this._translate.instant('group_saved'));
            this._router.navigate(['settings', 'group', this.group.name]);
        }, () => {
            this.loadingSave = false;
        });
      } else {
        this._groupService.createGroup(this.group).subscribe( wm => {
            this.loadingSave = false;
            this._toast.success('', this._translate.instant('group_saved'));
            this._router.navigate(['settings', 'group', this.group.name]);
        }, () => {
            this.loadingSave = false;
        });
      }
    }

    clickAddAdminButton(username: string): void {
      this.loadingSave = true;
      this._groupService.addUserAdmin(this.group.name, username).subscribe( wm => {
          this.loadingSave = false;
          this._toast.success('', this._translate.instant('group_add_admin_saved'));
          this.reloadData(this.group.name);
      }, () => {
          this.loadingSave = false;
      });
    }

    clickRemoveAdminButton(username: string): void {
      this.loadingSave = true;
      this._groupService.removeUserAdmin(this.group.name, username).subscribe( wm => {
          this.loadingSave = false;
          this._toast.success('', this._translate.instant('group_remove_admin_saved'));
          this.reloadData(this.group.name);
      }, () => {
          this.loadingSave = false;
      });
    }

    clickRemoveUserButton(username: string): void {
      this.loadingSave = true;
      this._groupService.removeUser(this.group.name, username).subscribe( wm => {
          this.loadingSave = false;
          this._toast.success('', this._translate.instant('group_remove_user_saved'));
          this.reloadData(this.group.name);
      }, () => {
          this.loadingSave = false;
      });
    }

    clickAddUserButton(): void {
      this.loadingSave = true;
      this._groupService.addUser(this.group.name, this.addUserUsername).subscribe(() => {
          this.loadingSave = false;
          this._toast.success('', this._translate.instant('group_add_user_saved'));
          this.reloadData(this.group.name);
      }, () => {
          this.loadingSave = false;
      });
    }
}
