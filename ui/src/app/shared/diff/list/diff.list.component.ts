import { ChangeDetectionStrategy, Component, Input } from '@angular/core';
import { Mode } from 'app/shared/diff/item/diff.item.component';

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
    styleUrls: ['./diff.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
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
