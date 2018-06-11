
import {map} from 'rxjs/operators';
import { Injectable } from '@angular/core';
import {HttpClient, HttpHeaders} from '@angular/common/http';
import {Operation, PerformAsCodeResponse} from '../../model/operation.model';
import {Observable} from 'rxjs';

@Injectable()
export class ImportAsCodeService {


    constructor(private _http: HttpClient) { }

    import(key: string, ope: Operation): Observable<Operation> {
        return this._http.post<Operation>('/import/' + key, ope);
    }

    create(key: string, uuid: string): Observable<PerformAsCodeResponse> {
        return this._http.post<PerformAsCodeResponse>('/import/' + key + '/' + uuid + '/perform', null,
            {observe: 'response'}).pipe(map(res => {
            let headers: HttpHeaders = res.headers;
            let resp = new PerformAsCodeResponse();
            resp.workflowName = headers.get('X-Api-Workflow-Name');
            resp.msgs = res.body;
            return resp;
        }));
    }
}
