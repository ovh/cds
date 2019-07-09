import { ChangeDetectionStrategy, Component, Input } from '@angular/core';
import { SuiPopupConfig } from '@richardlt/ng2-semantic-ui';
import { Warning } from 'app/model/warning.model';

@Component({
    selector: 'app-warning-mark-list',
    templateUrl: './warning.mark.list.html',
    styleUrls: ['./warning.mark.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WarningMarkListComponent {

    @Input() warnings: Array<Warning>;

    constructor(private _globalConfig: SuiPopupConfig) {
        this._globalConfig.isBasic = false;
    }

}
