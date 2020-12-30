import { ChangeDetectionStrategy, Component, EventEmitter, Input, Output } from '@angular/core';

@Component({
    selector: 'app-select-filter',
    templateUrl: './select.filter.html',
    styleUrls: ['./select.filter.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class SelectFilterComponent {

    _value: string;
    @Input() set value(d: string) {
        this._value = d;
        if (d && this.options) {
            if (this.options.findIndex( o => o === d) === -1) {
                this.options.push(d);
            }
        }
    }
    get value() {
        return this._value;
    }
    _options: Array<string>;
    @Input() set options(d: Array<string>) {
        if (d && this.value) {
            if (d.findIndex( o => o === this.value) === -1) {
                d.push(this.value);
            }
        }
        this._options = d;
    }
    get options() {
        return this._options;
    }
    @Input() searchable = true;
    @Input() disabled = false;
    @Output() valueChange = new EventEmitter();

    constructor() {
    }

    filterOptions = (opts: Array<string>, query: string): Array<string> => {
        if (!query || query === '') {
            return opts;
        }
        this.value = query;
        this.valueChange.emit(this.value);

        let result = Array<string>();
        opts.forEach(o => {
            if (o.indexOf(query) > -1) {
                result.push(o);
            }
        });
        if (result.indexOf(query) === -1) {
            result.push(query);
        }
        return result;
    };
}
