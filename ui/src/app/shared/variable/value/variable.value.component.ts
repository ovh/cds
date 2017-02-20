import {Component, Input, EventEmitter, OnInit} from '@angular/core';
import {SharedService} from '../../shared.service';
import {Output} from '@angular/core/src/metadata/directives';

@Component({
    selector: 'app-variable-value',
    templateUrl: './variable.value.html',
    styleUrls: ['./variable.value.scss']
})
export class VariableValueComponent implements OnInit {

    @Input() type: string;
    @Input() value: string|number|boolean;
    @Input() disabled: boolean;

    @Output() valueChange = new EventEmitter<string|number|boolean>();
    @Output() valueUpdating = new EventEmitter<boolean>();

    constructor(public _sharedService: SharedService) {
    }

    ngOnInit(): void {
        if (this.type === 'boolean') {
            this.value = (this.value === 'true');
        }
    }

    valueChanged(): void {
        this.valueChange.emit(this.value);
    }

    sendValueChanged(): void {
        this.valueUpdating.emit(true);
    }

}
