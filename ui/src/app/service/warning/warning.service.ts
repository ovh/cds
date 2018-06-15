import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {Warning} from '../../model/warning.model';

@Injectable()
export class WarningService {
    constructor(private _http: HttpClient) {}

    getProjectWarnings(key: string): Observable<Array<Warning>> {
        return this._http.get<Array<Warning>>('/warning/' + key);
    }
}
