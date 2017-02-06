import {Injectable} from '@angular/core';
import {Http, URLSearchParams, RequestOptions} from '@angular/http';
import {Observable} from 'rxjs/Rx';
import {Branch} from '../../model/repositories.model';

declare var _: any;
@Injectable()
export class ApplicationWorkflowService {

    constructor(private _http: Http) {
    }

    /**
     * Get the list of branch for the application
     * @param key Project unique key
     * @param appName Application Name
     * @returns {Observable<Array<Branch>>}
     */
    getBranches(key: string, appName: string): Observable<Array<string>> {
        return this._http.get('/project/' + key + '/application/' + appName + '/branches').map(res => {
            let data: Array<Branch> = res.json();
            let branches: Array<string> = new Array<string>();
            data.forEach(b => {
                branches.push(b.display_id);
            });
            branches = _.sortBy(branches);
            return branches.reverse();
        });
    }

    /**
     * Get the list of version for the given branch
     * @param key Project Unique key
     * @param appName Application Name
     * @param branchName branch name
     * @returns {Observable<Array<number>>}
     */
    getVersions(key: string, appName: string, branchName: string): Observable<Array<number>> {
        let options = new RequestOptions();
        options.search = new URLSearchParams();
        options.search.set('branch', branchName);
        return this._http.get('/project/' + key + '/application/' + appName + '/version', options).map(res => res.json());
    }
}
