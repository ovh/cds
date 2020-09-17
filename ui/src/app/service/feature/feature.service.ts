import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { FeatureEnabledResponse } from 'app/model/feature.model';
import { Observable } from 'rxjs';

@Injectable()
export class FeatureService {
    constructor(private _http: HttpClient) {
    }

    isEnabled(name: string, params: { [key: string]: string; }): Observable<FeatureEnabledResponse> {
        return this._http.post<FeatureEnabledResponse>(`/feature/enabled/${name}`, params);
    }
}
