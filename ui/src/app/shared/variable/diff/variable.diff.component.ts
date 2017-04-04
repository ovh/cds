import {Component, Input} from '@angular/core';
import {Variable} from '../../../model/variable.model';

@Component({
    selector: 'app-variable-diff',
    templateUrl: './variable.diff.html',
    styleUrls: ['./variable.diff.scss']
})
export class VariableDiffComponent {

    @Input() type: string;
    @Input() variableBefore: Variable;
    @Input() variableAfter: Variable;

    constructor() { }
}
