import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {VariableAudit} from '../../model/variable.model';
/**
 * Service to access Project from API.
 * Only used by ProjectStore
 */
@Injectable()
export class ProjectAuditService {


    constructor(private _http: HttpClient) {
    }

    getVariableAudit(key: string, varName: string): Observable<Array<VariableAudit>> {
        return this._http.get<Array<VariableAudit>>('/project/' + key + '/variable/' + varName + '/audit');
    }
}


