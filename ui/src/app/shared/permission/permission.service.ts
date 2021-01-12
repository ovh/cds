import {Injectable} from '@angular/core';

@Injectable()
export class PermissionService {
    private r = 4;
    private rx = 5;
    private rwx = 7;

    private permissions = [
        { name: 'permission_read', value: this.r },
        { name: 'permission_read_execute', value: this.rx },
        { name: 'permission_read_write_execute', value: this.rwx }
    ];

    /**
     * Get ReadWriteExecture permission code
     *
     * @returns
     */
    getRWX(): number {
      return this.rwx;
    }

    /**
     * Get permissions list
     *
     * @returns
     */
    getPermissions() {
        return this.permissions;
    }
}
