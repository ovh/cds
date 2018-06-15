import {Injectable} from '@angular/core';
import * as immutable from 'immutable';
import {BehaviorSubject, Observable} from 'rxjs';
import {Warning} from '../../model/warning.model';
import {WarningService} from './warning.service';

/**
 * Service to get warnings
 */
@Injectable()
export class WarningStore {

    // List of all warnings.
    private _projectWarning: BehaviorSubject<immutable.Map<string, immutable.Map<string, Warning>>> =
        new BehaviorSubject(immutable.Map<string, immutable.Map<string, Warning>>());

    constructor(private _warningService: WarningService) {
    }

    getProjectWarnings(key: string) {
        if (this._projectWarning.getValue().size === 0) {
            this._warningService.getProjectWarnings(key).subscribe(ws => {
                this.pushWarnings(key, ws);
            });
        }
        return new Observable<immutable.Map<string, immutable.Map<string, Warning>>>(fn => this._projectWarning.subscribe(fn));
    }

    pushWarnings(key: string, ws: Array<Warning>): void {
        let projWarnings = immutable.Map<string, Warning>();
        if (ws) {
            ws.forEach(w => {
                if (w.key) {
                    projWarnings = projWarnings.set(w.type + '-' + w.element, w)
                }
            });
            this._projectWarning.next(this._projectWarning.getValue().set(key, projWarnings));
        }
    }
}
