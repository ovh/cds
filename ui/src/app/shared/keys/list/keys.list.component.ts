import {Component, EventEmitter, Input, Output} from '@angular/core';
import {Table} from '../../table/table';
import {PermissionValue} from '../../../model/permission.model';
import {KeyEvent} from '../key.event';
import {Key} from '../../../model/keys.model';

@Component({
    selector: 'app-keys-list',
    templateUrl: './keys.list.html',
    styleUrls: ['./keys.list.scss']
})
export class KeysListComponent extends Table {

    @Input() keys: Array<Key>;
    @Input() loading: boolean;
    @Input() edit: number;
    @Output() deleteEvent = new EventEmitter<KeyEvent>();
    permission = PermissionValue;

    constructor() {
        super();
    }

    getData(): any[] {
        return this.keys;
    }

    deleteKey(k: Key): void {
        this.deleteEvent.emit(new KeyEvent('delete', k));
    }


}
