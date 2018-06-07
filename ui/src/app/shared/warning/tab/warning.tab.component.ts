import {Component, Input} from '@angular/core';
import {Warning} from '../../../model/warning.model';

@Component({
    selector: 'app-warning-tab',
    templateUrl: './warning.tab.html',
    styleUrls: ['./warning.tab.scss']
})
export class WarningTabComponent {

    @Input() warnings: Array<Warning>;
}
