import { ChangeDetectionStrategy, Component, EventEmitter, Input, Output } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { GroupPermission } from 'app/model/group.model';
import { PermissionEvent } from 'app/shared/permission/permission.event.model';
import { PermissionService } from 'app/shared/permission/permission.service';
import { Table } from 'app/shared/table/table';

@Component({
    selector: 'app-permission-list',
    templateUrl: './permission.list.html',
    styleUrls: ['./permission.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class PermissionListComponent extends Table<GroupPermission> {

    @Input() permissions: GroupPermission[];
    @Input() edit = false;

    // submit(project/app/pip view), form (wizard)
    @Input() mode = 'submit';

    @Output() event = new EventEmitter<PermissionEvent>();
    @Output() permissionChange = new EventEmitter<boolean>();

    private permissionsList;

    constructor(_permService: PermissionService, private _translate: TranslateService) {
        super();
        this.permissionsList = _permService.getPermissions();
    }

    getData(): Array<GroupPermission> {
        return this.permissions;
    }

    sendEvent(type: string, gp: GroupPermission): void {
        gp.updating = true;
        let gpEvent: PermissionEvent = new PermissionEvent(type, gp);
        this.event.emit(gpEvent);
    }

    getPermissionName(permValue: number): string {
        if (this.permissionsList) {
            let perm = this.permissionsList.find(p => p.value === permValue);
            if (perm) {
                return perm.name;
            }
        }
    }

    pushChange(): void {
        this.permissionChange.emit(true);
    }

    formatPermission() {
        let translate = this._translate;
        return function (event) {
            return translate.instant(event.name);
        };
    }
}
