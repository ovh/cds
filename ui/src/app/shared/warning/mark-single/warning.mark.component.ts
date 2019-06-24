import { ChangeDetectionStrategy, Component, Input } from '@angular/core';
import { Warning } from 'app/model/warning.model';

@Component({
    selector: 'app-warning-mark',
    templateUrl: './warning.mark.html',
    styleUrls: ['./warning.mark.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WarningMarkComponent {

    @Input() warning: Warning;
}
