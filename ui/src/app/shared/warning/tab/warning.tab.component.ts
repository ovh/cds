import { ChangeDetectionStrategy, Component, Input } from '@angular/core';
import { Warning } from 'app/model/warning.model';

@Component({
    selector: 'app-warning-tab',
    templateUrl: './warning.tab.html',
    styleUrls: ['./warning.tab.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WarningTabComponent {

    @Input() warnings: Array<Warning>;
}
