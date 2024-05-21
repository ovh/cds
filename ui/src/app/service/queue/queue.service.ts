import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { WorkflowNodeJobRun } from 'app/model/workflow.run.model';
import { Observable } from 'rxjs';
import { V2WorkflowRunJob } from '../../../../dist/workflow-graph/lib/v2.workflow.run.model';

@Injectable()
export class QueueService {
    constructor(private _http: HttpClient) { }

    getWorkflows(status: Array<string>): Observable<Array<WorkflowNodeJobRun>> {
        let params = new HttpParams();
        if (status && status.length > 0) {
            status.forEach(s => {
                params = params.append('status', s);
            });
        }
        return this._http.get<Array<WorkflowNodeJobRun>>('/queue/workflows', { params });
    }

    getJobInfos(jobID: number): Observable<WorkflowNodeJobRun> {
        return this._http.get<WorkflowNodeJobRun>(`/queue/workflows/${jobID}/infos`);
    }

    getV2Jobs(statuses: string[], regions: string[], offset: number, limit: number): Observable<any> {
        let params = new HttpParams();
        if (statuses) {
            statuses.forEach(s => {
                params = params.append("status", s);
            })
        }
        return this._http.get<any>(`/v2/queue?limit=${limit}&offset=${offset}`, { params: params, observe: 'response'});
    }
}
