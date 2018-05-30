import {Component, Input, OnInit} from '@angular/core';
import {Warning} from '../../../model/warning.model';

@Component({
    selector: 'app-warning-mark-list',
    templateUrl: './warning.mark.list.html',
    styleUrls: ['./warning.mark.list.scss']
})
export class WarningMarkListComponent {

    @Input() warnings: Array<Warning>;

}
