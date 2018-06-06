import {Component, Input} from '@angular/core';
import {Warning} from '../../../model/warning.model';
import {SuiPopupConfig} from 'ng2-semantic-ui';

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
