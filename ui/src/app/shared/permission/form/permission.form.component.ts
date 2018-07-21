import {Component, EventEmitter, Input, Output} from '@angular/core';
import {first} from 'rxjs/operators';
import {Group, GroupPermission} from '../../../model/group.model';
import {GroupService} from '../../../service/group/group.service';
import {PermissionEvent} from '../permission.event.model';
import {PermissionService} from '../permission.service';

@Component({
    selector: 'app-permission-form',
    templateUrl: './permission.form.html',
    styleUrls: ['./permission.form.scss']
})
export class PermissionFormComponent {

    public ready = false;


    permissionList: {};
    groupList: Group[];
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
        this._groupService.getGroups().pipe(first()).subscribe(groups => {
            this.groupList = groups;
            this.ready = true;
        });
    }
}
