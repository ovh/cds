import { Component, Input } from '@angular/core';

export type Selector = (d: any) => string;

export class Column {
    name: string;
    selector: Selector;
}

@Component({
    selector: 'app-sorted-table',
    templateUrl: './sorted-table.html',
    styleUrls: ['./sorted-table.scss']
})
export class SortedTableComponent {
    @Input() columns: Array<Column>;
    @Input() data: any;

    constructor() {
    }
}
