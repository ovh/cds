import { ChangeDetectionStrategy, ChangeDetectorRef, Component, ViewChild } from '@angular/core';
import { Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { AddProject } from 'app/store/project.action';
import { SemanticModalComponent } from 'ng-semantic/ng-semantic';
import { finalize, first } from 'rxjs/operators';
import { Group, GroupPermission } from '../../../model/group.model';
import { Project } from '../../../model/project.model';
import { GroupService } from '../../../service/group/group.service';
import { PermissionService } from '../../../shared/permission/permission.service';
import { ToastService } from '../../../shared/toast/ToastService';

@Component({
    selector: 'app-project-add',
    templateUrl: './project.add.html',
    styleUrls: ['./project.add.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectAddComponent {

    project: Project;
    newGroup: Group = new Group();
    group: Group = new Group();

    loading = false;
    nameError = false;
    keyError = false;
    fileTooLarge = false;

    groupList: Group[];

    @ViewChild('createGroupModal')
    modalCreateGroup: SemanticModalComponent;

    constructor(
        private _toast: ToastService,
        private _translate: TranslateService,
        private _router: Router,
        private _groupService: GroupService,
        private _permissionService: PermissionService,
        private store: Store,
        private _cd: ChangeDetectorRef
    ) {
        this.project = new Project();
        this.loadGroups(null);
    }

    /**
     * Generation of project key
     */
    generateKey(name: string) {
        if (!name) {
            this.project.key = '';
            return;
        }
        if (!this.project.key) {
            this.project.key = '';
        }
        this.project.key = name.toUpperCase();
        this.project.key = this.project.key.replace(/([.,; *`§%&#_\-'+?^=!:$\\"{}()|\[\]\/\\])/g, '').substr(0, 5);
    }

    /**
     * Create a new project
     */
    createProject(): void {
        this.loading = true;
        this.nameError = false;
        this.keyError = false;
        if (!this.project.name || this.project.name.length === 0) {
            this.nameError = true;
        }
        if (!this.project.key || this.project.key.length === 0) {
            this.keyError = true;
        }
        if (this.project.key) {
            let regexp = new RegExp('^[A-Z0-9]*$');
            if (!regexp.test(this.project.key)) {
                this.keyError = true;
            }
        }
        if (this.group && this.group.name !== '') {
            let gp = new GroupPermission();
            gp.permission = this._permissionService.getRWX();
            gp.group = this.group;
            this.project.groups = new Array<GroupPermission>();
            this.project.groups.push(gp);
        }

        if (!this.nameError && !this.keyError) {
            this.loading = true;
            this.store.dispatch(new AddProject(this.project))
                .pipe(finalize(() => {
                    this.loading = false;
                    this._cd.markForCheck();
                }))
                .subscribe(() => {
                    this._toast.success('', this._translate.instant('project_added'));
                    this._router.navigate(['/project', this.project.key]);
                });
        } else {
            this.loading = false;
        }
    }

    loadGroups(selected: string) {
        this._groupService.getAll(true)
            .pipe(first(), finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(groups => {
            this.groupList = groups;
            if (selected == null) {
                return;
            }
            this.group = groups.find(g => g.name === selected);
        });
    }

    setGroup(groupID): void {
        this.group = this.groupList.find(g => g.id === Number(groupID));
    }

    /**
     * Create a new group and add it to the project.
     */
    createGroup(): void {
        if (!this.newGroup.name && this.newGroup.name.length === 0) {
            return;
        }
        this.loading = true;
        this._groupService.create(this.newGroup)
            .pipe(finalize(() => {
                this.loading = false;
                this.newGroup = new Group();
                this._cd.markForCheck();
            }))
            .subscribe(() => {
            this._toast.success('', this._translate.instant('group_added'));
            this.loadGroups(this.newGroup.name);
            this.modalCreateGroup.hide();
        });
    }

    fileEvent(event: { content: string, file: File }) {
        this.fileTooLarge = event.file.size > 100000
        if (this.fileTooLarge) {
            return;
        }
        this.project.icon = event.content;
    }
}
