import { ChangeDetectionStrategy, Component, EventEmitter, Input, OnInit, Output } from '@angular/core';
import { SharedService } from 'app/shared/shared.service';

@Component({
    selector: 'app-variable-value',
    templateUrl: './variable.value.html',
    styleUrls: ['./variable.value.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class VariableValueComponent implements OnInit {
    _type: string;
    @Input() set type (t: string) {
        this._type = t;
        if (t === 'boolean') {
            this.value = this.value === 'true';
        }
    }
    get type(): string{
        return this._type;
    }
    @Input() value: string | number | boolean;
    @Input() disabled: boolean;
    @Output() valueChange = new EventEmitter<string | number | boolean>();
    @Output() valueUpdating = new EventEmitter<boolean>();

    constructor(
        public _sharedService: SharedService // used in html
    ) { }

    ngOnInit(): void {
        if (this.type === 'boolean') {
            this.value = (this.value === 'true');
        }
    }

    valueChanged(): void {
        this.valueChange.emit(this.value);
        this.valueUpdating.emit(true);
    }

    sendValueChanged(): void {
        this.valueUpdating.emit(true);
    }
}
