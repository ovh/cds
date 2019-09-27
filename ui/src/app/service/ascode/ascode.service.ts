import { Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import { Store } from '@ngxs/store';
import { ResyncEvents } from 'app/store/ascode.action';


@Injectable()
export class AuthenticationService {
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
    resyncPRAsCode(projectKey: string, repo: string): Observable<any> {
        let params = new HttpParams();
        params = params.append('repo', repo);
        return this._http.post(`/project/${projectKey}/ascode/events/resync`, null, {params: params})
            .map(() => {
            this._store.dispatch(new ResyncEvents());
        });
    }
}
