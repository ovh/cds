import {HttpClient, HttpParams} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Environment} from 'app/model/environment.model';
import { Operation } from 'app/model/operation.model';
import {Usage} from 'app/model/usage.model';
import {Observable} from 'rxjs';
/**
 * Service to access Environment Audit from API.
 */
@Injectable()
export class EnvironmentService {


    constructor(private _http: HttpClient) {
    }

    getEnvironment(key: string, envName: string): Observable<Environment> {
        let params = new HttpParams();
        params = params.append('withUsage', 'true');

        return this._http
            .get<Environment>(`/project/${key}/environment/${envName}`, { params })
    }

    get(key: string): Observable<Array<Environment>> {
        let params = new HttpParams();
        params = params.append('withUsage', 'true');

        return this._http.get<Array<Environment>>('/project/' + key + '/environment', {params});
    }

    getUsage(key: string, envName: string): Observable<Usage> {
        return this._http.get<Usage>('/project/' + key + '/environment/' + envName + '/usage');
    }

    /**
     * Update environment as code
     *
     * @param key Project key
     * @param environment Environment to update
     * @param branch Branch name to create the PR
     * @param message Message of the commit
     */
    updateAsCode(key: string, oldEnvName: string, environment: Environment, branch, message: string): Observable<Operation> {
        let params = new HttpParams();
        params = params.append('branch', branch);
        params = params.append('message', message)
        return this._http.put<Operation>(`/project/${key}/environment/${oldEnvName}/ascode`, environment, { params });
    }
}
