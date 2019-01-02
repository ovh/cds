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
    CONFIRM_BUTTON = 'confirm-button',
    LABEL = 'label',
}

export type SelectorType = <T>(d: T) => ColumnType;
export type Selector = <T>(d: T) => any;
export type Filter = (f: string) => (d: any) => any;

export class Column {
    type: ColumnType | SelectorType;
    name: string;
    class: string;
    selector: Selector;
    sortable: boolean;
    sortKey: string;
}

@Pipe({ name: 'selector' })
export class SelectorPipe<T> implements PipeTransform {
    transform(columns: Array<Column>, data: T): Array<any> {
        return columns.map(c => {
            let type: ColumnType;
            switch (typeof c.type) {
                case 'function':
                    type = <ColumnType>(c.type)(data);
                    break;
                default:
                    type = c.type;
                    break;
            }
            return {
                ...c,
                type,
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
export class DataTableComponent<T> extends Table<T> implements OnChanges {
    @Input() columns: Array<Column>;
    @Output() sortChange = new EventEmitter<string>();
    @Output() dataChange = new EventEmitter<number>();
    @Input() loading: boolean;
    @Input() withLineClick: boolean;
    @Output() clickLine = new EventEmitter<any>();

    @Input() data: Array<T>;
    @Input() withPagination: number;
    @Input() withFilter: Filter;

    sortedColumn: Column;
    sortedColumnDirection: direction;
    allData: Array<T>;
    dataForCurrentPage: any;
    pagesCount: number;
    filterFunc: Filter;
    filter: string;
    filteredData: Array<T>;
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

    getData(): Array<T> {
        this.filteredData = this.data;
        if (this.filter && this.filterFunc) {
            this.filteredData = this.data.filter(this.filterFunc(this.filter));
        }

        if (this.filteredData) {
            this.dataChange.emit(this.filteredData.length);
        }

        return this.filteredData;
    }

    getDataForCurrentPage(): Array<T> {
        this.pagesCount = this.getNbOfPages();
        if (this.pagesCount < this.currentPage) {
            this.currentPage = 1;
        }

        let data: Array<T>;
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
