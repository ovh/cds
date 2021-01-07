import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Store } from '@ngxs/store';
import { ResyncEvents } from 'app/store/ascode.action';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';

@Injectable()
export class AscodeService {
    constructor(
        private _http: HttpClient,
        private _store: Store
    ) { }

    resyncPRAsCode(projectKey: string, workflowName: string): Observable<boolean> {
        return this._http.post<boolean>(`/project/${projectKey}/workflows/${workflowName}/ascode/events/resync`, null)
            .pipe(map(() => {
                this._store.dispatch(new ResyncEvents());
                return true;
            }));
    }
}
