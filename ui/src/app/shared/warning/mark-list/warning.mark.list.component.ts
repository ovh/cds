import {Component, Input} from '@angular/core';
import {Warning} from '../../../model/warning.model';
import {SuiPopupConfig} from 'ng2-semantic-ui';

@Component({
    selector: 'app-warning-mark-list',
    templateUrl: './warning.mark.list.html',
    styleUrls: ['./warning.mark.list.scss']
})
export class WarningMarkListComponent {

    _warnings: Array<Warning>;
    @Input('warnings')
    set warnings(data: Array<Warning>) {
        this._warnings = data;
        if (this._warnings) {
            this.message = this._warnings.map(w => w.message).join('\n');
        }
    };
    get warnings() {
        return this._warnings;
    }

    message: string;

    constructor(private _globalConfig: SuiPopupConfig) {
        this._globalConfig.isBasic = false;
    }

}
