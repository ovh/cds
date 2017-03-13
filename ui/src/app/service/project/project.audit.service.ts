import {Injectable} from '@angular/core';
import {Http} from '@angular/http';
import {Observable} from 'rxjs/Rx';
import {VariableAudit} from '../../model/variable.model';
/**
 * Service to access Project from API.
 * Only used by ProjectStore
 */
@Injectable()
export class ProjectAuditService {


    constructor(private _http: Http) {
    }

    getVariableAudit(key: string, varName: string): Observable<Array<VariableAudit>> {
        return this._http.get('/project/' + key + '/variable/' + varName + '/audit').map(res => res.json());
    }
}


