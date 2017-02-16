import {Component, Input} from '@angular/core';

@Component({
    selector: 'app-status-icon',
    templateUrl: './status.icon.html',
    styleUrls: ['./status.icon.scss']
})
export class StatusIconComponent {

    @Input() status: string;
    @Input() value: string;

    constructor() { }
}
