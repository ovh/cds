import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Observable';
import {Branch, Remote} from '../../model/repositories.model';
import {HttpClient, HttpParams} from '@angular/common/http';

@Injectable()
export class ApplicationWorkflowService {

    constructor(private _http: HttpClient) {
    }

    /**
     * Get the list of branch for the application
     * @param key Project unique key
     * @param appName Application Name
     * @returns {Observable<Array<Branch>>}
     */
    getBranches(key: string, appName: string, remote: string = ''): Observable<Array<Branch>> {
        let params = new HttpParams();
        params = params.append('remote', remote);
        return this._http.get<Array<Branch>>('/project/' + key + '/application/' + appName + '/branches', {params});
    }

    /**
     * Get the list of remotes for the application
     * @param key Project unique key
     * @param appName Application Name
     * @returns {Observable<Array<Branch>>}
     */
    getRemotes(key: string, appName: string): Observable<Array<Remote>> {
        return this._http.get<Array<Remote>>('/project/' + key + '/application/' + appName + '/remotes');
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
