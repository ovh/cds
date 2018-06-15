import {Component, ViewChild} from '@angular/core';
import {Router} from '@angular/router';
import {TranslateService} from '@ngx-translate/core';
import {cloneDeep} from 'lodash';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';
import {first} from 'rxjs/operators';
import {Group, GroupPermission} from '../../../model/group.model';
import {Key, KeyType} from '../../../model/keys.model';
import {Project} from '../../../model/project.model';
import {GroupService} from '../../../service/group/group.service';
import {ProjectStore} from '../../../service/project/project.store';
import {PermissionService} from '../../../shared/permission/permission.service';
import {ToastService} from '../../../shared/toast/ToastService';

@Component({
    selector: 'app-project-add',
    templateUrl: './project.add.html',
    styleUrls: ['./project.add.scss']
})
export class ProjectAddComponent {

    project: Project;
    newGroup: Group = new Group();
    group: Group = new Group();
    addSshKey = false;
    sshKey: Key;

    loading = false;
    nameError = false;
    keyError = false;
    sshError = false;

    groupList: Group[];

    @ViewChild('createGroupModal')
    modalCreateGroup: SemanticModalComponent;

    constructor(private _projectStore: ProjectStore, private _toast: ToastService, private _translate: TranslateService,
                private _router: Router, private _groupService: GroupService, private _permissionService: PermissionService) {
        this.project = new Project();
        this.sshKey = new Key();
        this.sshKey.type = KeyType.SSH;
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
        this.sshKey.name = 'sshkey';
    }

    /**
     * Create a new project
     */
    createProject(): void {
        this.loading = true;
        this.nameError = false;
        this.keyError = false;
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
        if (this.group && this.group.name !== '') {
          let gp = new GroupPermission();
          gp.permission = this._permissionService.getRWX();
          gp.group = this.group;
          this.project.groups = new Array<GroupPermission>();
          this.project.groups.push(gp);
        }

        if (this.addSshKey && (!this.sshKey.name || this.sshKey.name === '')) {
            this.sshError = true;
        }

        if (!this.nameError && !this.keyError && !this.sshError) {
            if (this.addSshKey) {
                this.project.keys = new Array<Key>();

                let sshKeyCloned = cloneDeep(this.sshKey);
                if (sshKeyCloned.name.indexOf('proj-') !== 0) {
                    sshKeyCloned.name = 'proj-' + sshKeyCloned.name;
                }
                this.project.keys.push(sshKeyCloned);
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

    loadGroups(selected: string) {
        this._groupService.getGroups(true).pipe(first()).subscribe(groups => {
            this.groupList = groups;
            this.loading = false;
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
        this._groupService.createGroup(this.newGroup).subscribe(() => {
            this._toast.success('', this._translate.instant('group_added'));
            this.loadGroups(this.newGroup.name);
            this.modalCreateGroup.hide();
            this.loading = false;
        }, () => {
            this.loading = false;
            this.newGroup = new Group();
        });
    }
}
