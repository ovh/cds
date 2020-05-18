import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Store } from '@ngxs/store';
import { ResyncEvents } from 'app/store/ascode.action';
import { Observable } from 'rxjs';

@Injectable()
export class AscodeService {
    constructor(
        private _http: HttpClient,
        private _store: Store
    ) { }

    resyncPRAsCode(projectKey: string, workflowName: string): Observable<boolean> {
        return this._http.post<boolean>(`/project/${projectKey}/workflows/${workflowName}/ascode/events/resync`, null)
            .map(() => {
                this._store.dispatch(new ResyncEvents());
                return true;
            });
    }
}
