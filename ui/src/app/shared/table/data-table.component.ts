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
    BUTTON = 'button',
    CONFIRM_BUTTON = 'confirm-button',
    LABEL = 'label',
}

export type SelectorType<T> = (d: T) => ColumnType;
export type Selector<T> = (d: T) => any;
export type Filter<T> = (f: string) => (d: T) => boolean;
export type Select<T> = (d: T) => boolean;

export class Column<T> {
    type: ColumnType | SelectorType<T>;
    name: string;
    class: string;
    selector: Selector<T>;
    sortable: boolean;
    sortKey: string;
}

@Pipe({ name: 'selector' })
export class SelectorPipe<T> implements PipeTransform {
    transform(columns: Array<Column<T>>, data: T): Array<any> {
        return columns.map(c => {
            let type: ColumnType;
            switch (typeof c.type) {
                case 'function':
                    type = (<SelectorType<T>>c.type)(data);
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

@Pipe({ name: 'select' })
export class SelectPipe<T extends WithKey> implements PipeTransform {
    transform(selected: Array<string>, data: T): boolean {
        return !!selected.find(s => s === data.key());
    }
}

export interface WithKey {
    key(): string;
}

@Component({
    selector: 'app-data-table',
    templateUrl: './data-table.html',
    styleUrls: ['./data-table.scss']
})
export class DataTableComponent<T extends WithKey> extends Table<T> implements OnChanges {
    @Input() columns: Array<Column<T>>;
    @Output() sortChange = new EventEmitter<string>();
    @Output() dataChange = new EventEmitter<number>();
    @Input() loading: boolean;
    @Input() withLineClick: boolean;
    @Output() clickLine = new EventEmitter<T>();
    @Output() selectChange = new EventEmitter<Array<string>>();
    @Input() withSelect: boolean | Select<T>;
    selected: Object = {};
    @Input() data: Array<T>;
    @Input() withPagination: number;
    @Input() withFilter: Filter<T>;
    sortedColumn: Column<T>;
    sortedColumnDirection: direction;
    allData: Array<T>;
    dataForCurrentPage: any;
    pagesCount: number;
    filterFunc: Filter<T>;
    filter: string;
    filteredData: Array<T>;
    indexSelected: number;

    ngOnChanges() {
        this.allData = this.data;

        if (this.withSelect && this.allData) {
            this.allData.forEach(d => { this.selected[d.key()] = false });
            if (typeof this.withSelect === 'function') {
                this.allData.filter(<Select<T>>this.withSelect).forEach(d => this.selected[d.key()] = true);
                this.emitSelectChange();
            }
        }

        this.nbElementsByPage = this.withPagination;
        this.filterFunc = this.withFilter;
        this.getDataForCurrentPage();
    }

    columnClick(event: Event, c: Column<T>) {
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

    lineClick(i: number, d: T) {
        if (this.withLineClick) {
            this.indexSelected = i;
            this.clickLine.emit(d);
        }
    }

    onSelectChange(e: any, key: string) {
        this.selected[key] = e;
        this.emitSelectChange();
    }

    emitSelectChange() {
        this.selectChange.emit(Object.keys(this.selected).filter(k => this.selected[k]));
    }
}
