import {Injectable} from '@angular/core';
import {Http} from '@angular/http';
import {Observable} from 'rxjs/Rx';
import {VariableAudit} from '../../model/variable.model';
/**
 * Service to access Environment Audit from API.
 */
@Injectable()
export class EnvironmentAuditService {


    constructor(private _http: Http) {
    }

    getVariableAudit(key: string, envName: string, varName: string): Observable<Array<VariableAudit>> {
        return this._http.get('/project/' + key + '/environment/' + envName + '/variable/' + varName + '/audit').map(res => res.json());
    }
}


