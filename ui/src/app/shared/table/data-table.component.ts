import { Component, EventEmitter, Input, Output } from '@angular/core';
import { Table } from './table';

type direction = string;
const ASC: direction = 'asc';
const DESC: direction = 'desc';

export enum ColumnType {
    TEXT = 'text',
    ICON = 'icon',
    LINK = 'link',
    ROUTER_LINK = 'router-link',
    MARKDOWN = 'markdown'
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
    selector: 'app-data-table',
    templateUrl: './data-table.html',
    styleUrls: ['./data-table.scss']
})
export class DataTableComponent extends Table {
    @Input() columns: Array<Column>;
    @Output() sortChange = new EventEmitter<string>();
    @Output() dataChange = new EventEmitter<number>();
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
    filteredData: any;

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
        this.filteredData = this.data;
        if (this.filter && this.filterFunc) {
            this.filteredData = this.data.filter(this.filterFunc(this.filter));
        }

        if (this.filteredData) {
            this.dataChange.emit(this.filteredData.length);
        }

        return this.filteredData;
    }

    getDataForCurrentPage(): any[] {
        this.pagesCount = this.getNbOfPages();
        if (this.pagesCount < this.currentPage) {
            this.currentPage = 1;
        }

        let data: any[];
        if (!this.withPagination) {
            data = this.getData();
        } else {
            data = super.getDataForCurrentPage();
        }
        this.dataForCurrentPage = data;

        return data;
    }

    filterChange() {
        this.getDataForCurrentPage();
    }

    pageChange(n: number) {
        this.goTopage(n);
    }
}
