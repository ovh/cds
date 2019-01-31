
import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';
import { Application, Vulnerability } from '../../model/application.model';
import { Key } from '../../model/keys.model';
import { Variable } from '../../model/variable.model';

@Injectable()
export class ApplicationService {

    constructor(private _http: HttpClient) {
    }

    /**
     * Get the given application from API
     * @param key Project unique key
     * @param appName Application Name
     */
    getApplication(key: string, appName: string): Observable<Application> {
        let params = new HttpParams();
        params = params.append('withNotifs', 'true');
        params = params.append('withUsage', 'true');
        params = params.append('withIcon', 'true');
        params = params.append('withKeys', 'true');
        params = params.append('withDeploymentStrategies', 'true');
        params = params.append('withVulnerabilities', 'true');

        return this._http.get<Application>('/project/' + key + '/application/' + appName, {params: params}).pipe(map(a => {
            a.vcs_strategy.password = '**********';
            return a;
        }));
    }

    /**
     * Update the given application
     * @param key Project unique key
     * @param application Application to update
     * @returns {Observable<Application>}
     */
    updateApplication(key: string, appOldName: string, app: Application): Observable<Application> {
        return this._http.put<Application>('/project/' + key + '/application/' + appOldName, app);
    }

    /**
     * Clone application
     * @param key Project unique key
     * @param appName Application to clone
     * @param application Application data
     * @returns {Observable<Application>}
     */
    cloneApplication(key: string, appName: string, application: Application): Observable<Application> {
        return this._http.post<Application>('/project/' + key + '/application/' + appName + '/clone', application);
    }

    /**
     * Create application
     * @param key Project unique key
     * @param application Application data
     * @returns {Observable<Application>}
     */
    createApplication(key: string, application: Application): Observable<Application> {
        return this._http.post<Application>('/project/' + key + '/applications', application);
    }


    /**
     * Delete an application
     * @param key Project unique key
     * @param appName Application name to delete
     * @returns {Observable<boolean>}
     */
    deleteApplication(key: string, appName: string): Observable<boolean> {
        return this._http.delete('/project/' + key + '/application/' + appName).pipe(map(res => {
            return true;
        }));
    }

    /**
     * Remove the given repository from the given application
     * @param key Project unique key
     * @param appName Application name
     * @param repoManName Repo manager name
     * @returns {Observable<Application>}
     */
    removeRepository(key: string, appName: string, repoManName: string): Observable<Application> {
        let url = '/project/' + key + '/repositories_manager/' + repoManName + '/application/' + appName + '/detach';
        return this._http.post<Application>(url, null);
    }

    /**
     * Connect the given repository to the application
     * @param key Project unique key
     * @param appName Application name
     * @param repoManName Repository manager name
     * @param repoFullName Repository fullname
     * @returns {Observable<Application>}
     */
    connectRepository(key: string, appName: string, repoManName: string, repoFullName: string): Observable<Application> {
        let url = '/project/' + key + '/repositories_manager/' + repoManName + '/application/' + appName + '/attach';
        let headers = new HttpHeaders();
        headers.append('Content-Type', 'application/x-www-form-urlencoded');
        let params = new HttpParams();
        params = params.append('fullname', repoFullName);
        return this._http.post<Application>(url, params.toString(), {headers: headers, params: params});
    }

    /**
     * Add a variable in the application
     * @param key Project unique key
     * @param appName Application name
     * @param v Variable to add
     * @returns {Observable<Application>}
     */
    addVariable(key: string, appName: string, v: Variable): Observable<Application> {
        let url = '/project/' + key + '/application/' + appName + '/variable/' + v.name;
        return this._http.post<Application>(url, v);
    }

    /**
     * Update a variable in the application
     * @param key Project unique key
     * @param appName Application name
     * @param v Variable to update
     * @returns {Observable<Application>}
     */
    updateVariable(key: string, appName: string, v: Variable): Observable<Application> {
        let url = '/project/' + key + '/application/' + appName + '/variable/' + v.name;
        return this._http.put<Application>(url, v);
    }

    /**
     * Delete a variable from the application
     * @param key Project unique key
     * @param appName Application name
     * @param v Variable to delete
     * @returns {Observable<Application>}
     */
    removeVariable(key: string, appName: string, v: Variable): Observable<Application> {
        let url = '/project/' + key + '/application/' + appName + '/variable/' + v.name;
        return this._http.delete<Application>(url);
    }

    /**
     * Add a key
     * @param key Project unique key
     * @param appName Application name
     * @param k Key to add
     */
    addKey(key: string, appName: string, k: Key): Observable<Key> {
        return this._http.post<Key>('/project/' + key + '/application/' + appName + '/keys', k);
    }

    /**
     * Remove a key
     * @param key Project unique key
     * @param appName Application name
     * @param kName Key to remove
     */
    removeKey(key: string, appName: string, kName: string): Observable<any> {
        return this._http.delete('/project/' + key + '/application/' + appName + '/keys/' + kName);
    }

    /**
     * Save a deployment strategy
     * @param key Project unique key
     * @param appName Application name
     * @param pfName Integration name
     * @param pfConfig integration config
     */
    saveDeploymentStrategy(key: string, appName: string, pfName: string, pfConfig: any): Observable<Application> {
        let url = '/project/' + key + '/application/' + appName + '/deployment/config/' + pfName;
        return this._http.post<Application>(url, pfConfig);
    }

    /**
     * Delete a deployment strategy
     * @param key Project unique key
     * @param appName Application name
     * @param pfName Integration name
     */
    deleteDeploymentStrategy(key: string, appName: string, pfName: string): Observable<Application> {
        let url = '/project/' + key + '/application/' + appName + '/deployment/config/' + pfName;
        return this._http.delete<Application>(url);
    }

     /**
     * Get application deployment strategies
     * @param key Project unique key
     * @param appName Application name
     */
    getDeploymentStrategies(key: string, appName: string): Observable<Map<string, any>> {
        let url = '/project/' + key + '/application/' + appName + '/deployment/config';
        return this._http.get<Map<string, any>>(url);
    }

    /**
     * Ignore vulnerability
     * @param key project unique key
     * @param appName application name
     * @param id identifiant of the vulnerability
     */
    ignoreVulnerability(key: string, appName: string, v: Vulnerability): Observable<Vulnerability> {
        let url = '/project/' + key + '/application/' + appName + '/vulnerability/' + v.id;
        return this._http.post<Vulnerability>(url, v);
    }
}
