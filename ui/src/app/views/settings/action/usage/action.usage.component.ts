import { Component, Input } from '@angular/core';
import { Usage } from '../../../../model/action.model';

@Component({
    selector: 'app-action-usage',
    templateUrl: './action.usage.html',
    styleUrls: ['./action.usage.scss']
})
export class ActionUsageComponent {
    @Input() usage: Usage;

    constructor() { }
}
