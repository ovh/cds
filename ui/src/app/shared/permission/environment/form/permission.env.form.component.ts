import {Component, EventEmitter, Input, Output} from '@angular/core';
import {Project} from '../../../../model/project.model';
import {GroupService} from '../../../../service/group/group.service';
import {Group, GroupPermission} from '../../../../model/group.model';
import {PermissionService} from '../../permission.service';
import {EnvironmentPermissionEvent} from '../../permission.event.model';
import {first} from 'rxjs/operators';

@Component({
    selector: 'app-permission-env-form',
    templateUrl: './permission.env.form.html',
    styleUrls: ['./permission.env.form.scss']
})
export class PermissionEnvironmentFormComponent {

    @Input() project: Project;
    @Input() loading = false;
    @Output() addEnvPermEvent = new EventEmitter<EnvironmentPermissionEvent>();

    groups: Array<Group>;
    perms: {};

    selectedGroups = new Array<string>();
    selectedEnvs = new Array<string>();
    selectedPerm: string;

    constructor(private _groupService: GroupService, private _permissionService: PermissionService) {
        this._groupService.getGroups().pipe(first()).subscribe(gs => {
            this.groups = gs;
        });
        this.perms = this._permissionService.getPermissions();
    }



    saveEnvPermissions(): void {
        if (!this.selectedPerm || this.selectedGroups.length === 0 || this.selectedEnvs.length === 0) {
            return;
        }

        this.selectedEnvs.forEach(eIDString => {
            let env = this.project.environments.find(environment => environment.id === Number(eIDString)) ;
            if (!env) {
                return;
            }
            // Filter groups that are not already in the environment
            let newGroups: Array<Group> = this.selectedGroups.map(ids => {
                if (env.groups) {
                    let isHere = env.groups.find(gp => gp.group.id === Number(ids));
                    if (isHere) {
                        return;
                    }
                }
               return this.groups.find(g => g.id === Number(ids));
            });

            let gps = newGroups.map(g => {
                let groupPermissions = new GroupPermission();
                groupPermissions.permission = Number(this.selectedPerm);
                groupPermissions.group = g;
                return groupPermissions;
            });
            this.addEnvPermEvent.emit(new EnvironmentPermissionEvent('add', env, gps));
        });
        delete this.selectedEnvs;
        delete this.selectedGroups;
        delete this.selectedPerm;
    }
}
