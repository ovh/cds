
import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Application, Vulnerability } from 'app/model/application.model';
import { Key } from 'app/model/keys.model';
import { Operation } from 'app/model/operation.model';
import { Observable } from 'rxjs';

@Injectable()
export class ApplicationService {

    constructor(private _http: HttpClient) {
    }

    getApplication(key: string, appName: string): Observable<Application> {
        let params = new HttpParams();
        params = params.append('withNotifs', 'true');
        params = params.append('withUsage', 'true');
        params = params.append('withIcon', 'true');
        params = params.append('withKeys', 'true');
        params = params.append('withDeploymentStrategies', 'true');
        params = params.append('withVulnerabilities', 'true');
        return this._http.get<Application>(
            `/project/${key}/application/${appName}`,
            { params }
        )
    }

    /**
     * Add a key
     *
     * @param key Project unique key
     * @param appName Application name
     * @param k Key to add
     */
    addKey(key: string, appName: string, k: Key): Observable<Key> {
        return this._http.post<Key>('/project/' + key + '/application/' + appName + '/keys', k);
    }

    /**
     * Get application deployment strategies
     *
     * @param key Project unique key
     * @param appName Application name
     */
    getDeploymentStrategies(key: string, appName: string): Observable<Map<string, any>> {
        let url = '/project/' + key + '/application/' + appName + '/deployment/config';
        return this._http.get<Map<string, any>>(url);
    }

    /**
     * Ignore vulnerability
     *
     * @param key project unique key
     * @param appName application name
     * @param id identifiant of the vulnerability
     */
    ignoreVulnerability(key: string, appName: string, v: Vulnerability): Observable<Vulnerability> {
        let url = '/project/' + key + '/application/' + appName + '/vulnerability/' + v.id;
        return this._http.post<Vulnerability>(url, v);
    }

    /**
     * Update application as code
     *
     * @param key Project key
     * @param application Application to update
     * @param branch Branch name to create the PR
     * @param message Message of the commit
     */
    updateAsCode(key: string, oldAppName, application: Application, branch, message: string): Observable<Operation> {
        let params = new HttpParams();
        params = params.append('branch', branch);
        params = params.append('message', message)
        return this._http.put<Operation>(`/project/${key}/application/${oldAppName}/ascode`, application, { params });
    }
}
