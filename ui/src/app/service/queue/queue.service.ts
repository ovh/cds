import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { WorkflowNodeJobRun } from 'app/model/workflow.run.model';
import { Observable } from 'rxjs';

@Injectable()
export class QueueService {
    constructor(private _http: HttpClient) { }

    getWorkflows(): Observable<Array<WorkflowNodeJobRun>> {
        return this._http.get<Array<WorkflowNodeJobRun>>('/queue/workflows');
    }

    getJobInfos(jobID: number): Observable<WorkflowNodeJobRun> {
        return this._http.get<WorkflowNodeJobRun>(`/queue/workflows/${jobID}/infos`);
    }
}
