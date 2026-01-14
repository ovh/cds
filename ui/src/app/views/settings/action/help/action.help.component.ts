import { ChangeDetectionStrategy, Component } from '@angular/core';

@Component({
    standalone: false,
    selector: 'app-action-help',
    templateUrl: './action.help.html',
    styleUrls: ['./action.help.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ActionHelpComponent { }
