import {Injectable} from '@angular/core';

@Injectable()
export class PermissionService {

    private permissions = [
        { 'name': 'permission_read', 'value': 4 },
        { 'name': 'permission_read_execute', 'value': 5 },
        { 'name': 'permission_read_write_execute', 'value': 7 }
    ];

    /**
     * Get permissions list
     * @returns {number}
     */
    getPermissions() {
        return this.permissions;
    }
}
