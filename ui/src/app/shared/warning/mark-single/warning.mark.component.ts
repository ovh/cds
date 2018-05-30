import {Component, Input, OnInit} from '@angular/core';
import {Warning} from '../../../model/warning.model';

@Component({
    selector: 'app-warning-mark',
    templateUrl: './warning.mark.html',
    styleUrls: ['./warning.mark.scss']
})
export class WarningMarkComponent {

    @Input() warning: Warning;
}
