import {Component, Input} from '@angular/core';
import {SuiPopupConfig} from 'ng2-semantic-ui';
import {Warning} from '../../../model/warning.model';

@Component({
    selector: 'app-warning-mark-list',
    templateUrl: './warning.mark.list.html',
    styleUrls: ['./warning.mark.list.scss']
})
export class WarningMarkListComponent {

    @Input() warnings: Array<Warning>;

    constructor(private _globalConfig: SuiPopupConfig) {
        this._globalConfig.isBasic = false;
    }

}
