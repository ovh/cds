import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Observable';
import {VariableAudit} from '../../model/variable.model';
import {HttpClient} from '@angular/common/http';
/**
 * Service to access Application Audit from API.
 */
@Injectable()
export class ApplicationAuditService {


    constructor(private _http: HttpClient) {
    }

    getVariableAudit(key: string, appName: string, varName: string): Observable<Array<VariableAudit>> {
        return this._http.get<Array<VariableAudit>>('/project/' + key + '/application/' + appName + '/variable/' + varName + '/audit');
    }
}


