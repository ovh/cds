import { ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { AuthenticationState } from 'app/store/authentication.state';
import { finalize } from 'rxjs/operators';
import { Group } from '../../../../model/group.model';
import { User } from '../../../../model/user.model';
import { GroupService } from '../../../../service/group/group.service';
import { UserService } from '../../../../service/user/user.service';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { ToastService } from '../../../../shared/toast/ToastService';

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
    private groupname: string;
    private groupnamePattern: RegExp = new RegExp('^[a-zA-Z0-9._-]{1,}$');
    groupPatternError = false;
    path: Array<PathItem>;

    constructor(
        private _userService: UserService,
        private _groupService: GroupService,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _route: ActivatedRoute,
        private _router: Router,
        private _store: Store,
        private _cd: ChangeDetectorRef
    ) {
        this.currentUser = this._store.selectSnapshot(AuthenticationState.user);
        this._userService.getUsers().subscribe(users => {
            this.users = users;
            this._cd.markForCheck();
        });
    }

    ngOnInit() {
        this._route.params.subscribe(params => {
            if (params['groupname'] !== 'add') {
                this.reloadData(params['groupname']);
            } else {
                this.group = new Group();
                this.updatePath();
            }
            this._cd.markForCheck();
        });
    }

    reloadData(groupname: string): void {
        this._groupService.getByName(groupname).subscribe(grp => {
            this.group = grp;
            this.groupname = grp.name;
            if (grp.members) {
                for (let i = 0; i < grp.members.length; i++) {
                    if (this.currentUser.username === grp.members[i].username) {
                        this.currentUserIsAdminOnGroup = true;
                        break;
                    }
                }
            }
            this.updatePath();
            this._cd.markForCheck();
        });
    }

    clickDeleteButton(): void {
        this.deleteLoading = true;
        this._groupService.delete(this.group.name)
            .pipe(
                finalize(() => {
                    this.deleteLoading = false;
                    this._cd.markForCheck();
                })
            )
            .subscribe(wm => {
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
            this._groupService.update(this.groupname, this.group)
                .pipe(
                    finalize(() => {
                        this.loading = false;
                        this._cd.markForCheck();
                    })
                )
                .subscribe(wm => {
                    this._toast.success('', this._translate.instant('group_saved'));
                    this._router.navigate(['settings', 'group', this.group.name]);
                });
        } else {
            this._groupService.create(this.group)
                .pipe(
                    finalize(() => {
                        this.loading = false;
                        this._cd.markForCheck();
                    })
                )
                .subscribe(wm => {
                    this._toast.success('', this._translate.instant('group_saved'));
                    this._router.navigate(['settings', 'group', this.group.name]);
                });
        }
    }

    clickAddAdminButton(username: string): void {
        this.loading = true;
        this._groupService.addAdmin(this.group.name, username)
            .pipe(
                finalize(() => {
                    this.loading = false;
                    this._cd.markForCheck();
                })
            )
            .subscribe(wm => {
                this._toast.success('', this._translate.instant('group_add_admin_saved'));
                this.reloadData(this.group.name);
            });
    }

    clickRemoveAdminButton(username: string): void {
        this.loading = true;
        this._groupService.removeAdmin(this.group.name, username)
            .pipe(
                finalize(() => {
                    this.loading = false;
                    this._cd.markForCheck();
                })
            )
            .subscribe(wm => {
                this._toast.success('', this._translate.instant('group_remove_admin_saved'));
                this.reloadData(this.group.name);
            });
    }

    clickRemoveUserButton(username: string): void {
        this.loading = true;
        this._groupService.removeMember(this.group.name, username)
            .pipe(
                finalize(() => {
                    this.loading = false;
                    this._cd.markForCheck();
                })
            )
            .subscribe(wm => {
                this._toast.success('', this._translate.instant('group_remove_user_saved'));
                this.reloadData(this.group.name);
            });
    }

    clickAddUserButton(): void {
        this.loading = true;
        this._groupService.addMember(this.group.name, this.addUserUsername)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(() => {
            this._toast.success('', this._translate.instant('group_add_user_saved'));
            this.reloadData(this.group.name);
        });
    }

    updatePath() {
        this.path = [<PathItem>{
            translate: 'common_settings'
        }, <PathItem>{
            translate: 'group_list_title',
            routerLink: ['/', 'settings', 'group']
        }];

        if (this.group) {
            if (this.group.id) {
                this.path.push(<PathItem>{
                    text: this.group.name,
                    routerLink: ['/', 'settings', 'group', this.group.name]
                });
            } else {
                this.path.push(<PathItem>{
                    translate: 'common_create'
                });
            }
        }
    }
}
