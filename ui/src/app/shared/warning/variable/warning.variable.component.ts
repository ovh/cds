import {Component, Input} from '@angular/core';
import {Warning} from '../../../model/warning.model';

@Component({
    selector: 'app-warning-variable',
    templateUrl: './warning.variable.html',
    styleUrls: ['./warning.variable.scss']
})
export class WarningVariableComponent {

    @Input() warning: Warning;
}
