import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Observable';
import {Environment} from '../../model/environment.model';
import {HttpClient, HttpParams} from '@angular/common/http';
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
}
