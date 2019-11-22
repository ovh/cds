import { ChangeDetectionStrategy, Component, EventEmitter, Input, Output } from '@angular/core';
import { Key } from 'app/model/keys.model';
import { Warning } from 'app/model/warning.model';
import { KeyEvent } from 'app/shared/keys/key.event';
import { Table } from 'app/shared/table/table';

@Component({
    selector: 'app-keys-list',
    templateUrl: './keys.list.html',
    styleUrls: ['./keys.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class KeysListComponent extends Table<Key> {
    @Input() keys: Array<Key>;
    @Input() loading: boolean;
    @Input() edit: boolean;
    @Input() warnings: Map<string, Warning>;
    @Output() deleteEvent = new EventEmitter<KeyEvent>();

    constructor() {
        super();
    }

    getData(): Array<Key> {
        return this.keys;
    }

    deleteKey(k: Key): void {
        this.deleteEvent.emit(new KeyEvent('delete', k));
    }
}
