import { Component, EventEmitter, Input, Output } from '@angular/core';
import { Table } from './table';

type direction = string;
const ASC: direction = 'asc';
const DESC: direction = 'desc';

export enum ColumnType {
    TEXT = 'text',
    HTML = 'html',
    LINK = 'link',
    ROUTER_LINK = 'router-link'
}

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
export class SortedTableComponent extends Table {
    @Input() columns: Array<Column>;
    @Input() data: any;
    @Output() sortChange = new EventEmitter<any>();
    @Input() loading: boolean;
    @Input() withFilter: boolean;

    @Input()
    set withPagination(n: number) {
        this.nbElementsByPage = n;
    }
    get withPagination() { return this.nbElementsByPage; }

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
        super();
    }

    getData(): any[] {
        return this.data;
    }

    getDataForCurrentPage(): any[] {
        if (!this.withPagination) {
            return this.getData();
        }
        return super.getDataForCurrentPage();
    }
}
