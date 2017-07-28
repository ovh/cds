import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Rx';
import {Branch} from '../../model/repositories.model';
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
    getBranches(key: string, appName: string): Observable<Array<Branch>> {
        return this._http.get('/project/' + key + '/application/' + appName + '/branches');
    }

    /**
     * Get the list of version for the given branch
     * @param key Project Unique key
     * @param appName Application Name
     * @param branchName branch name
     * @returns {Observable<Array<number>>}
     */
    getVersions(key: string, appName: string, branchName: string): Observable<Array<number>> {
        let params = new HttpParams();
        params = params.append('branch', branchName);
        return this._http.get('/project/' + key + '/application/' + appName + '/version', {params: params});
    }
}
