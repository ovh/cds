import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { VCSInfos } from 'app/model/repositories.model';
import { Observable } from 'rxjs';

@Injectable()
export class ApplicationWorkflowService {

    constructor(private _http: HttpClient) {
    }

    /**
     * Get the list of branch for the application
     *
     * @param key Project unique key
     * @param appName Application Name
     * @param remote Remote Name
     * @returns
     */
    getVCSInfos(key: string, appName: string, remote?: string): Observable<VCSInfos> {
        let params = new HttpParams();
        if (remote) {
            params = params.append('remote', remote);
        }
        return this._http.get<VCSInfos>('/project/' + key + '/application/' + appName + '/vcsinfos', {params});
    }
}
