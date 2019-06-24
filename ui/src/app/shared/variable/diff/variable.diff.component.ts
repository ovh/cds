import { ChangeDetectionStrategy, Component, Input } from '@angular/core';
import { Variable } from 'app/model/variable.model';

@Component({
    selector: 'app-variable-diff',
    templateUrl: './variable.diff.html',
    styleUrls: ['./variable.diff.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class VariableDiffComponent {

    @Input() type: string;
    @Input() variableBefore: Variable;
    @Input() variableAfter: Variable;

    constructor() { }
}
