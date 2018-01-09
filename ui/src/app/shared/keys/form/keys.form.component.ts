import {Component, EventEmitter, Input, OnInit, Output} from '@angular/core';
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
    @Output() keyEvent = new EventEmitter<KeyEvent>();

    constructor() {
        this.newKey = new Key();
    }

    ngOnInit(): void {
        this.newKey.type = this.keyTypes[0];
    }

    addKey(): void {
        this.keyEvent.emit(new KeyEvent('add', this.newKey));
    }
}
