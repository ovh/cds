import { Component, EventEmitter, Input, OnChanges, Output, Pipe, PipeTransform, } from '@angular/core';
import { Table } from './table';

type direction = string;
const ASC: direction = 'asc';
const DESC: direction = 'desc';

export enum ColumnType {
    TEXT = 'text',
    ICON = 'icon',
    LINK = 'link',
    ROUTER_LINK = 'router-link',
    MARKDOWN = 'markdown',
    DATE = 'date',
    CONFIRM_BUTTON = 'confirm-button'
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

@Pipe({ name: 'selector' })
export class SelectorPipe implements PipeTransform {
    transform(columns: Array<Column>, data: any): Array<any> {
        return columns.map(c => {
            return {
                type: c.type,
                selector: c.selector(data)
            };
        });
    }
}

@Component({
    selector: 'app-data-table',
    templateUrl: './data-table.html',
    styleUrls: ['./data-table.scss']
})
export class DataTableComponent extends Table implements OnChanges {
    @Input() columns: Array<Column>;
    @Output() sortChange = new EventEmitter<string>();
    @Output() dataChange = new EventEmitter<number>();
    @Input() loading: boolean;
    @Input() withLineClick: boolean;
    @Output() clickLine = new EventEmitter<any>();

    @Input() data: any;
    @Input() withPagination: number;
    @Input() withFilter: Filter;

    sortedColumn: Column;
    sortedColumnDirection: direction;
    allData: any;
    dataForCurrentPage: any;
    pagesCount: number;
    filterFunc: Filter;
    filter: string;
    filteredData: any;
    indexSelected: number;

    ngOnChanges() {
        this.allData = this.data;
        this.nbElementsByPage = this.withPagination;
        this.filterFunc = this.withFilter;
        this.getDataForCurrentPage();
    }

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

    lineClick(i: number, d: any) {
        if (this.withLineClick) {
            this.indexSelected = i;
            this.clickLine.emit(d);
        }
    }
}
