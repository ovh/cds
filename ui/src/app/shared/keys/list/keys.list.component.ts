import {Component, EventEmitter, Input, Output} from '@angular/core';
import {Table} from '../../table/table';
import {Project} from '../../../model/project.model';
import {PermissionValue} from '../../../model/permission.model';
import {KeyEvent} from '../key.event';
import {Key} from '../../../model/keys.model';

@Component({
    selector: 'app-keys-list',
    templateUrl: './keys.list.html',
    styleUrls: ['./keys.list.scss']
})
export class KeysListComponent extends Table {

    @Input() project: Project;
    @Input() loading: boolean;
    @Output() deleteEvent = new EventEmitter<KeyEvent>();
    permission = PermissionValue;

    constructor() {
        super();
    }

    getData(): any[] {
        return this.project.keys;
    }

    deleteKey(k: Key): void {
        this.deleteEvent.emit(new KeyEvent('delete', k));
    }


}
