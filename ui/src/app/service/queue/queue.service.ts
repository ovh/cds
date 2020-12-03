import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { WorkflowNodeJobRun } from 'app/model/workflow.run.model';
import { Observable } from 'rxjs';

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
}
