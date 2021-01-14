import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { AuthenticationState } from 'app/store/authentication.state';
import { finalize } from 'rxjs/operators';
import { Group, GroupMember } from '../../../../model/group.model';
import { AuthentifiedUser, AuthSummary } from '../../../../model/user.model';
import { GroupService } from '../../../../service/group/group.service';
import { UserService } from '../../../../service/user/user.service';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { ToastService } from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-group-edit',
    templateUrl: './group.edit.html',
    styleUrls: ['./group.edit.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class GroupEditComponent implements OnInit {
    loading = false;
    deleteLoading = false;
    groupName: string;
    group: Group;
    currentAuthSummary: AuthSummary;
    currentUserIsAdminOnGroup: boolean;
    addUserUsername: string;
    users: Array<AuthentifiedUser>;
    private groupnamePattern = new RegExp('^[a-zA-Z0-9._-]{1,}$');
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
        this.currentAuthSummary = this._store.selectSnapshot(AuthenticationState.summary);
    }

    ngOnInit() {
        this._route.params.subscribe(params => {
            if (params['groupname'] !== 'add') {
                this.groupName = params['groupname']
                this.loadGroup();
            } else {
                this.group = new Group();
                this.currentUserIsAdminOnGroup = true;
                this.updatePath();
                this._cd.markForCheck();
            }
        });

        this.getUsers();
    }

    getUsers(): void {
        this._userService.getUsers().subscribe(users => {
            this.users = users;
            this._cd.markForCheck();
        });
    }

    loadGroup(): void {
        this._groupService.getByName(this.groupName).subscribe(grp => {
            this.group = grp;
            this.updateDataFromGroup();
            this.updatePath();
            this._cd.markForCheck();
        });
    }

    updateDataFromGroup(): void {
        if (this.group.members) {
            for (let i = 0; i < this.group.members.length; i++) {
                if (this.currentAuthSummary.user.username === this.group.members[i].username) {
                    this.currentUserIsAdminOnGroup = this.group.members[i].admin;
                    break;
                }
            }
        }
    }

    saveGroup(): void {
        if (!this.group.name) {
            return;
        }

        if (!this.groupnamePattern.test(this.group.name)) {
            this.groupPatternError = true;
            this._cd.markForCheck();
            return;
        }

        this.loading = true;
        this._cd.markForCheck();
        if (this.group.id > 0) {
            this._groupService.update(this.groupName, this.group)
                .pipe(
                    finalize(() => {
                        this.loading = false;
                        this._cd.markForCheck();
                    })
                )
                .subscribe(g => {
                    this._toast.success('', this._translate.instant('group_saved'));
                    this.group = g;
                    this.updateDataFromGroup();
                    this._cd.markForCheck();
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
                .subscribe(() => {
                    this._toast.success('', this._translate.instant('group_saved'));
                    this._router.navigate(['settings', 'group', this.group.name]);
                });
        }
    }

    clickDelete(): void {
        this.deleteLoading = true;
        this._cd.markForCheck();
        this._groupService.delete(this.group.name)
            .pipe(
                finalize(() => {
                    this.deleteLoading = false;
                    this._cd.markForCheck();
                })
            )
            .subscribe(() => {
                this._toast.success('', this._translate.instant('group_deleted'));
                this._router.navigate(['settings', 'group']);
            });
    }

    clickAddMember(): void {
        this.loading = true;
        this._cd.markForCheck();
        this._groupService.addMember(this.group.name, <GroupMember>{
            username: this.addUserUsername,
            admin: false
        })
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(g => {
                this._toast.success('', this._translate.instant('group_add_user_saved'));
                this.group = g;
                this.updateDataFromGroup();
                this._cd.markForCheck();
            });
    }

    clickRemoveMember(username: string): void {
        this.loading = true;
        this._cd.markForCheck();
        this._groupService.removeMember(this.group.name, <GroupMember>{
            username
        })
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(g => {
                this._toast.success('', this._translate.instant('group_remove_user_saved'));
                if (username === this.currentAuthSummary.user.username) {
                    this._router.navigate(['settings', 'group']);
                    return;
                }
                this.group = g;
                this.updateDataFromGroup();
                this._cd.markForCheck();
            });
    }

    clickSetAdmin(username: string): void {
        this.loading = true;
        this._cd.markForCheck();
        this._groupService.updateMember(this.groupName, <GroupMember>{
            username,
            admin: true
        })
            .pipe(
                finalize(() => {
                    this.loading = false;
                    this._cd.markForCheck();
                })
            )
            .subscribe(g => {
                this.group = g;
                this.updateDataFromGroup();
                this._cd.markForCheck();
                this._toast.success('', this._translate.instant('group_add_admin_saved'));
            });
    }

    clickUnsetAdmin(username: string): void {
        this.loading = true;
        this._cd.markForCheck();
        this._groupService.updateMember(this.groupName, <GroupMember>{
            username,
            admin: false
        }).pipe(
            finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            })
        ).subscribe(g => {
            this.group = g;
            this.updateDataFromGroup();
            this._cd.markForCheck();
            this._toast.success('', this._translate.instant('group_remove_admin_saved'));
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
