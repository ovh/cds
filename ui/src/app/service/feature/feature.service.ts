import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { FeatureEnabledResponse } from 'app/model/feature.model';
import { Observable } from 'rxjs';

export enum FeatureNames {
    WorkflowRetentionPolicy = 'workflow-retention-policy',
    WorkflowRetentionMaxRuns = 'workflow-retention-maxruns',
    WorkflowV3 = 'workflow-v3'
}

@Injectable()
export class FeatureService {
    constructor(
        private _http: HttpClient
    ) { }

    isEnabled(name: FeatureNames, params: { [key: string]: string; }): Observable<FeatureEnabledResponse> {
        return this._http.post<FeatureEnabledResponse>(`/feature/enabled/${name}`, params);
    }
}
