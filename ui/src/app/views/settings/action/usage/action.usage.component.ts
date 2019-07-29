import { ChangeDetectionStrategy, Component, Input } from '@angular/core';
import { Usage } from '../../../../model/action.model';

@Component({
    selector: 'app-action-usage',
    templateUrl: './action.usage.html',
    styleUrls: ['./action.usage.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ActionUsageComponent {
    @Input() usage: Usage;

    constructor() { }
}
