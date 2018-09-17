import { Component, EventEmitter, Input, Output } from '@angular/core';
import { Table } from './table';

type direction = string;
const ASC: direction = 'asc';
const DESC: direction = 'desc';

export enum ColumnType {
    TEXT = 'text',
    ICON = 'icon',
    LINK = 'link',
    ROUTER_LINK = 'router-link'
}

export type Selector = (d: any) => any;
export type Filter = (f: string) => (d: any) => any;

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
    @Output() sortChange = new EventEmitter<any>();
    @Input() loading: boolean;

    @Input() set data(d: any) {
        this.allData = d;
        this.getDataForCurrentPage();
    }
    get data() { return this.allData; }

    @Input() set withPagination(n: number) {
        this.nbElementsByPage = n;
        this.getDataForCurrentPage();
    }
    get withPagination() { return this.nbElementsByPage; }


    @Input() set withFilter(f: Filter) {
        this.filterFunc = f;
        this.getDataForCurrentPage();
    }
    get withFilter() { return this.filterFunc; }

    sortedColumn: Column;
    sortedColumnDirection: direction;
    allData: any;
    dataForCurrentPage: any;
    pagesCount: number;
    filterFunc: Filter;
    filter: string;

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
        if (this.filter && this.filter) {
            return this.data.filter(this.filterFunc(this.filter));
        }
        return this.data;
    }

    getDataForCurrentPage(): any[] {
        let data: any[];
        if (!this.withPagination) {
            data = this.getData();
        } else {
            data = super.getDataForCurrentPage();
        }
        this.dataForCurrentPage = data;
        this.pagesCount = this.getNbOfPages();

        if (this.pagesCount < this.currentPage) {
            this.currentPage = 1;
        }

        return data;
    }

    filterChange() {
        this.getDataForCurrentPage();
    }
}
