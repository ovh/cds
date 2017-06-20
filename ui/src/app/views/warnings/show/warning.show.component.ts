import {ChangeDetectorRef, Component} from '@angular/core';
import {ActivatedRoute} from '@angular/router';
import {WarningStore} from '../../../service/warning/warning.store';
import {WarningUI} from '../../../model/warning.model';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';
import {Subscription} from 'rxjs/Subscription';

@Component({
    selector: 'app-warning-show',
    templateUrl: './warning.show.html',
    styleUrls: ['./warning.show.scss']
})
@AutoUnsubscribe()
export class WarningShowComponent {

    warnings: WarningUI;
    warningsSubscription: Subscription;
    key: string;

    constructor(private _activatedRoute: ActivatedRoute, private _warningStore: WarningStore, private _cd: ChangeDetectorRef) {
        this._activatedRoute.queryParams.subscribe(q => {
            if (q['key']) {
                this.key = q['key'];
            }
        });
        this.warningsSubscription = this._warningStore.getWarnings().subscribe(ws => {
            this.warnings = ws.get(this.key);
        });
    }
}
