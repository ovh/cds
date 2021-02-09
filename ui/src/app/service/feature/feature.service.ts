import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { FeatureEnabledResponse } from 'app/model/feature.model';
import { Observable } from 'rxjs';

export enum FeatureNames {
    CDNJobLogs = 'cdn-job-logs',
    CDNArtifact = 'cdn-artifact',
    WorkflowRetentionPolicy = 'workflow-retention-policy',
    WorkflowRetentionMaxRuns = 'workflow-retention-maxruns'
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
