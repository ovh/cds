import {Component, Input, EventEmitter} from '@angular/core';
import {SharedService} from '../../shared.service';
import {Output} from '@angular/core/src/metadata/directives';

@Component({
    selector: 'app-variable-value',
    templateUrl: './variable.value.html',
    styleUrls: ['./variable.value.scss']
})
export class VariableValueComponent  {

    @Input() type: string;
    @Input() value: string|number|boolean;
    @Input() disabled: boolean;

    @Output() valueChange = new EventEmitter<string|number|boolean>();
    @Output() valueUpdating = new EventEmitter<boolean>();

    constructor(public _sharedService: SharedService) {
    }

    valueChanged(): void {
        this.valueChange.emit(this.value);
    }

    sendValueChanged(): void {
        this.valueUpdating.emit(true);
    }

}
