import { Component, EventEmitter, Input, Output } from '@angular/core';

type direction = string;
const ASC: direction = 'asc';
const DESC: direction = 'desc';

export type ColumnType = string;
export const Text: ColumnType = 'text';
export const HTML: ColumnType = 'html';
export const Link: ColumnType = 'link';

export type Selector = (d: any) => any;

export class Column {
    type: ColumnType;
    name: string;
    selector: Selector;
    sortable: boolean;
    sortKey: string;
}

@Component({
    selector: 'app-sorted-table',
    templateUrl: './sorted-table.html',
    styleUrls: ['./sorted-table.scss']
})
export class SortedTableComponent {
    @Input() columns: Array<Column>;
    @Input() data: any;
    @Output() sortChange = new EventEmitter<any>();
    @Input() loading: boolean;

    sortedColumn: Column;
    sortedColumnDirection: direction;

    columnClick(event: Event, c: Column) {
        if (!c.sortable) {
            return;
        }

        this.sortedColumn = c;
        if (!this.sortedColumnDirection) {
            this.sortedColumnDirection = ASC;
        } else {
            this.sortedColumnDirection = this.sortedColumnDirection === ASC ? DESC : ASC;
        }

        this.sortChange.emit(this.sortedColumn.sortKey + ':' + this.sortedColumnDirection);
    }

    constructor() {
    }
}
