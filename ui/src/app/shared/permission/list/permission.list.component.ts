import {Component, EventEmitter, Input, Output} from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import {Table} from '../../table/table';
import {GroupPermission} from '../../../model/group.model';
import {PermissionService} from '../permission.service';
import {PermissionEvent} from '../permission.event.model';
import {Warning} from '../../../model/warning.model';

@Component({
    selector: 'app-permission-list',
    templateUrl: './permission.list.html',
    styleUrls: ['./permission.list.scss']
})
export class PermissionListComponent extends Table {

    @Input() permissions: GroupPermission[];
    @Input() edit = false;

    // submit(project/app/pip view), form (wizard)
    @Input() mode = 'submit';
    @Input() warning: Map<string, Warning>;

    @Output() event = new EventEmitter<PermissionEvent>();

    private permissionsList;

    constructor(_permService: PermissionService, private _translate: TranslateService) {
        super();
        this.permissionsList = _permService.getPermissions();
    }

    getData(): any[] {
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

    formatPermission() {
        let translate = this._translate;
        return function(event) {
            return translate.instant(event.name);
        };
    }
}
