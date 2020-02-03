import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Store } from '@ngxs/store';
import { ResyncEvents } from 'app/store/ascode.action';
import { Observable } from 'rxjs';


@Injectable()
export class AscodeService {
    constructor(
        private _http: HttpClient,
        private _store: Store
    ) {
    }

    /**
     * Resync As Code PR
     * @param projectKey
     * @param repo
     */
    resyncPRAsCode(projectKey: string, appName: string, repo?: string): Observable<any> {
        let params = new HttpParams();
        if (repo) {
            params = params.append('repo', repo);
        }
        params = params.append('appName', appName);

        return this._http.post(`/project/${projectKey}/ascode/events/resync`, null, {params})
            .map(() => {
            this._store.dispatch(new ResyncEvents());
        });
    }
}
