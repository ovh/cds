import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { WorkflowRunV3 } from './workflowv3.model';

@Injectable()
export class WorkflowV3RunService {
    constructor(
        private _http: HttpClient
    ) { }

    getWorkflowRun(projectKey: string, workflowName: string, runNumber: number): Observable<WorkflowRunV3> {
        let params = new HttpParams();
        params = params.append('format', 'json');
        params = params.append('full', 'true');
        return this._http.get<WorkflowRunV3>(`/project/${projectKey}/workflowv3/${workflowName}/run/${runNumber}`, { params });
    }
}
