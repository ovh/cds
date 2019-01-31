import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { VCSInfos } from '../../model/repositories.model';

@Injectable()
export class ApplicationWorkflowService {

    constructor(private _http: HttpClient) {
    }

    /**
     * Get the list of branch for the application
     * @param key Project unique key
     * @param appName Application Name
     * @param remote Remote Name
     * @returns {Observable<VCSInfos>}
     */
    getVCSInfos(key: string, appName: string, remote?: string): Observable<VCSInfos> {
        let params = new HttpParams();
        if (remote) {
            params = params.append('remote', remote);
        }
        return this._http.get<VCSInfos>('/project/' + key + '/application/' + appName + '/vcsinfos', {params});
    }

    /**
     * Get the list of version for the given branch
     * @param key Project Unique key
     * @param appName Application Name
     * @param branchName branch name
     * @returns {Observable<Array<number>>}
     */
    getVersions(key: string, appName: string, branchName: string, remote: string = ''): Observable<Array<number>> {
        let params = new HttpParams();
        params = params.append('remote', remote);
        params = params.append('branch', branchName);
        return this._http.get<Array<number>>('/project/' + key + '/application/' + appName + '/version', {params});
    }
}
