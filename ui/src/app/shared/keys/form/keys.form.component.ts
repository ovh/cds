import {Component, EventEmitter, Input, OnInit, Output} from '@angular/core';
import {cloneDeep} from 'lodash';
import {Key, KeyType} from '../../../model/keys.model';
import {KeyEvent} from '../key.event';

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
    @Input() defaultType = 'ssh';
    @Output() keyEvent = new EventEmitter<KeyEvent>();

    constructor() {
        this.newKey = new Key();
    }

    ngOnInit(): void {
        this.newKey.type = this.defaultType;
    }

    addKey(): void {
        let k = cloneDeep(this.newKey);
        if (k.name.indexOf(this.prefix) !== 0) {
            k.name = this.prefix + k.name;
        }
        this.keyEvent.emit(new KeyEvent('add', k));
    }
}
