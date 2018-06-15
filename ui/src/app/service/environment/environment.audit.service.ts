import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {VariableAudit} from '../../model/variable.model';
/**
 * Service to access Environment Audit from API.
 */
@Injectable()
export class EnvironmentAuditService {


    constructor(private _http: HttpClient) {
    }

    getVariableAudit(key: string, envName: string, varName: string): Observable<Array<VariableAudit>> {
        return this._http.get<Array<VariableAudit>>('/project/' + key + '/environment/' + envName + '/variable/' + varName + '/audit');
    }
}


