import { Key } from 'app/model/keys.model';

export class KeyEvent {
    type: string;
    key: Key;

    constructor(t: string, k: Key) {
        this.type = t;
        this.key = k;
    }
}
