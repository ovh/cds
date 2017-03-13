import {Injectable} from '@angular/core';
import {Http} from '@angular/http';
import {Observable} from 'rxjs/Rx';
import {VariableAudit} from '../../model/variable.model';
/**
 * Service to access Application Audit from API.
 */
@Injectable()
export class ApplicationAuditService {


    constructor(private _http: Http) {
    }

    getVariableAudit(key: string, appName: string, varName: string): Observable<Array<VariableAudit>> {
        return this._http.get('/project/' + key + '/application/' + appName + '/variable/' + varName + '/audit').map(res => res.json());
    }
}


