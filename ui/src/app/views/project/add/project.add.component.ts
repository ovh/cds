import {Component, ViewChild} from '@angular/core';
import {Project} from '../../../model/project.model';
import {PermissionEvent} from '../../../shared/permission/permission.event.model';
import {GroupPermission, Group} from '../../../model/group.model';
import {ProjectStore} from '../../../service/project/project.store';
import {ToastService} from '../../../shared/toast/ToastService';
import {TranslateService} from 'ng2-translate';
import {Router} from '@angular/router';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';
import {GroupService} from '../../../service/group/group.service';
import {PermissionFormComponent} from '../../../shared/permission/form/permission.form.component';
import {Variable} from '../../../model/variable.model';
import {cloneDeep} from 'lodash';

@Component({
    selector: 'app-project-add',
    templateUrl: './project.add.html',
    styleUrls: ['./project.add.scss']
})
export class ProjectAddComponent {

    project: Project;
    newGroup: Group = new Group();
    addSshKey = false;
    sshKeyVar: Variable;

    loading = false;
    nameError = false;
    keyError = false;
    groupError = false;
    sshError = false;

    @ViewChild('createGroupModal')
    modalCreateGroup: SemanticModalComponent;

    @ViewChild('permForm')
    permissionFormComponent: PermissionFormComponent;

    constructor(private _projectStore: ProjectStore, private _toast: ToastService, private _translate: TranslateService,
                private _router: Router, private _groupService: GroupService) {
        this.project = new Project();
        this.sshKeyVar = new Variable();
        this.sshKeyVar.type = 'key';
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
     * Manage permission events
     * @param event
     */
    permissionManagement(event: PermissionEvent): void {
        switch (event.type) {
            case 'add':
                if (!this.project.groups) {
                    this.project.groups = new Array<GroupPermission>();
                }
                event.gp.updating = false;

                let indexToAdd = this.project.groups.findIndex(gp => gp.group.name === event.gp.group.name);
                if (indexToAdd !== -1) {
                    return;
                }
                this.project.groups.push(cloneDeep(event.gp));
                break;
            case 'delete':
                let indexToDelete = this.project.groups.findIndex(gp => gp.group.name === event.gp.group.name);
                if (indexToDelete === -1) {
                    return;
                }
                this.project.groups.splice(indexToDelete, 1);
                break;
        }
    }

    /**
     * Create a new project
     */
    createProject(): void {
        this.loading = true;
        this.nameError = false;
        this.keyError = false;
        this.groupError = false;
        this.sshError = false;
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
        if (!this.project.groups || this.project.groups.length === 0) {
            this.groupError = true;
        }
        if (this.project.groups) {
            let index777 = this.project.groups.findIndex(gp => gp.permission === 7);
            if (index777 === -1) {
                this.groupError = true;
            }
        }

        if (this.addSshKey && (!this.sshKeyVar.name || this.sshKeyVar.name === '')) {
            this.sshError = true;
        }

        if (!this.nameError && !this.keyError && !this.groupError && !this.sshError) {

            if (this.addSshKey) {
                this.project.variables = new Array<Variable>();
                this.project.variables.push(this.sshKeyVar);
            }

            this._projectStore.createProject(this.project).subscribe(p => {
                this.loading = true;
                this._toast.success('', this._translate.instant('project_added'));
                this._router.navigate(['/project', p.key]);
            }, () => {
                this.loading = false;
            });
        } else {
            this.loading = false;
        }
    }

    /**
     * Create a new group and add it to the project.
     */
    createGroup(): void {
        if (!this.newGroup.name && this.newGroup.name.length === 0) {
            return;
        }
        this.modalCreateGroup.hide();
        this._groupService.createGroup(this.newGroup).subscribe(() => {
            this._toast.success('', this._translate.instant('group_added'));
            if (this.permissionFormComponent) {
                this.permissionFormComponent.loadGroups();
            }
            let gp = new GroupPermission();
            gp.permission = 7;
            gp.group = this.newGroup;
            let event = new PermissionEvent('add', gp);
            this.permissionManagement(event);
        });
        this.newGroup = new Group();
    }
}
