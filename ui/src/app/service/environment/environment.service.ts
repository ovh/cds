import {HttpClient, HttpParams} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Environment} from 'app/model/environment.model';
import {Usage} from 'app/model/usage.model';
import {Observable} from 'rxjs';
/**
 * Service to access Environment Audit from API.
 */
@Injectable()
export class EnvironmentService {


    constructor(private _http: HttpClient) {
    }

    get(key: string): Observable<Array<Environment>> {
        let params = new HttpParams();
        params = params.append('withUsage', 'true');

        return this._http.get<Array<Environment>>('/project/' + key + '/environment', {params});
    }

    getUsage(key: string, envName: string): Observable<Usage> {
        return this._http.get<Usage>('/project/' + key + '/environment/' + envName + '/usage');
    }
}
