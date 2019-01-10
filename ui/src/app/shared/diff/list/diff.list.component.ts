import { Component, Input } from '@angular/core';
import { Mode } from '../item/diff.item.component';

export class Item {
    name: string;
    translate: string;
    translateData: any;
    before: string;
    after: string;
    type: string;
}

@Component({
    selector: 'app-diff-list',
    templateUrl: './diff.list.html',
    styleUrls: ['./diff.list.scss']
})
export class DiffListComponent {
    mode: Mode = Mode.UNIFIED;
    @Input() items: Array<Item>;

    constructor() { }

    setUnified() {
        this.mode = Mode.UNIFIED;
    }

    setSplit() {
        this.mode = Mode.SPLIT;
    }
}
