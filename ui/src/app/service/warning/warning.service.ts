import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Warning} from 'app/model/warning.model';
import {Observable} from 'rxjs';

@Injectable()
export class WarningService {
    constructor(private _http: HttpClient) {}

    getProjectWarnings(key: string): Observable<Array<Warning>> {
        return this._http.get<Array<Warning>>('/warning/' + key);
    }

    update(key: string, w: Warning): Observable<Warning> {
        return this._http.put<Warning>('/warning/' + key + '/' + w.hash, w);
    }
}
