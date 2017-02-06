import {Component, Output, EventEmitter, Input} from '@angular/core';
import {GroupPermission, Group} from '../../../model/group.model';
import {PermissionService} from '../permission.service';
import {GroupService} from '../../../service/group/group.service';
import {PermissionEvent} from '../permission.event.model';

@Component({
    selector: 'app-permission-form',
    templateUrl: './permission.form.html',
    styleUrls: ['./permission.form.scss']
})
export class PermissionFormComponent {

    private ready = false;

    private permissionList;
    private groupList: Group[];
    newGroupPermission: GroupPermission;

    @Input() loading = false;

    // submit (green)/button (blue)
    @Input() buttonType = 'submit';

    @Output() createGroupPermissionEvent = new EventEmitter<PermissionEvent>();

    constructor(_permService: PermissionService, private _groupService: GroupService) {
        this.newGroupPermission = new GroupPermission();
        this.permissionList = _permService.getPermissions();
        this.loadGroups();
    }

    create(): void {
        this.newGroupPermission.permission = Number(this.newGroupPermission.permission); // select return a string
        let gpEvent: PermissionEvent = new PermissionEvent('add', this.newGroupPermission);
        this.createGroupPermissionEvent.emit(gpEvent);
    }

    loadGroups() {
        this._groupService.getGroups().first().subscribe( groups => {
            this.groupList = groups;
            this.ready = true;
        });
    }
}
