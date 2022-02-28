import { ChangeDetectionStrategy, Component, EventEmitter, Input, Output } from '@angular/core';
import { GroupPermission } from 'app/model/group.model';
import { PermissionEvent } from 'app/shared/permission/permission.event.model';
import { PermissionService } from 'app/shared/permission/permission.service';

@Component({
    selector: 'app-workflow-permission-form',
    templateUrl: './workflow-permission.form.html',
    styleUrls: ['./workflow-permission.form.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowPermissionFormComponent {
    permissionList: {};
    newGroupPermission: GroupPermission = new GroupPermission();

    @Input() loading = false;
    @Input() groups: Array<GroupPermission> = [];

    // submit (green)/button (blue)
    @Input() buttonType = 'submit';

    @Output() createGroupPermissionEvent = new EventEmitter<PermissionEvent>();

    constructor(
        private _permService: PermissionService
    ) {
        this.permissionList = this._permService.getPermissions();
    }

    create(): void {
        this.newGroupPermission.permission = Number(this.newGroupPermission.permission); // select return a string
        let gpEvent: PermissionEvent = new PermissionEvent('add', this.newGroupPermission);
        this.createGroupPermissionEvent.emit(gpEvent);
    }
}
