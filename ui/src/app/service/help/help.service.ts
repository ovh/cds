import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Help } from 'app/model/help.model';
import { Observable } from 'rxjs';

@Injectable()
export class HelpService {

    constructor(private _http: HttpClient) { }

    getHelp(): Observable<Help> {
        return this._http.get<Help>('/help');
    }
}
