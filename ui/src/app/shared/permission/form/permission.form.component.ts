import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, Output } from '@angular/core';
import { Group, GroupPermission } from 'app/model/group.model';
import { GroupService } from 'app/service/group/group.service';
import { PermissionEvent } from 'app/shared/permission/permission.event.model';
import { PermissionService } from 'app/shared/permission/permission.service';
import { finalize, first } from 'rxjs/operators';

@Component({
    selector: 'app-permission-form',
    templateUrl: './permission.form.html',
    styleUrls: ['./permission.form.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
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

    constructor(_permService: PermissionService, private _groupService: GroupService, private _cd: ChangeDetectorRef) {
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
        this._groupService.getGroups().pipe(first(), finalize(() => this._cd.markForCheck())).subscribe(groups => {
            this.groupList = groups;
            this.ready = true;
        });
    }
}
