import {Component, EventEmitter, Input, OnInit, Output} from '@angular/core';
import {Key, KeyType} from '../../../model/keys.model';
import {KeyEvent} from '../key.event';
import {cloneDeep} from 'lodash';

@Component({
    selector: 'app-keys-form',
    templateUrl: './keys.form.html',
    styleUrls: ['./keys.form.scss']
})
export class KeysFormComponent implements OnInit {

    newKey: Key;
    keyTypes = KeyType.values();

    @Input() loading = false;
    @Input() prefix: string;
    @Output() keyEvent = new EventEmitter<KeyEvent>();

    constructor() {
        this.newKey = new Key();
    }

    ngOnInit(): void {
        this.newKey.type = this.keyTypes[0];
    }

    addKey(): void {
        let k = cloneDeep(this.newKey);
        if (k.name.indexOf(this.prefix) !== 0) {
            k.name = this.prefix + k.name;
        }
        this.keyEvent.emit(new KeyEvent('add', k));
    }
}
