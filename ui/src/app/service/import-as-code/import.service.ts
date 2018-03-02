import { Injectable } from '@angular/core';
import {HttpClient} from '@angular/common/http';
import {Operation} from '../../model/operation.model';
import {Observable} from 'rxjs/Observable';

@Injectable()
export class ImportAsCodeService {

    constructor(private _http: HttpClient) { }

    import(key: string, ope: Operation): Observable<Operation> {
        return this._http.post<Operation>('/import/' + key, ope);
    }
}
