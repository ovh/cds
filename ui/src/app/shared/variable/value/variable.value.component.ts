import { ChangeDetectionStrategy, Component, EventEmitter, Input, OnInit, Output } from '@angular/core';
import { SharedService } from 'app/shared/shared.service';

@Component({
    selector: 'app-variable-value',
    templateUrl: './variable.value.html',
    styleUrls: ['./variable.value.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class VariableValueComponent implements OnInit {
    @Input() type: string;
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
