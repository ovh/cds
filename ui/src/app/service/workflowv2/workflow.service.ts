import { Injectable } from "@angular/core";
import { HttpClient, HttpParams } from "@angular/common/http";
import { Observable } from "rxjs";
import { CDNLogLinks } from "../../model/pipeline.model";
import { V2WorkflowRun, V2WorkflowRunJob, WorkflowRunInfo, WorkflowRunResult } from "../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";

@Injectable()
export class V2WorkflowRunService {
    constructor(
        private _http: HttpClient
    ) { }

    getRun(projKey: string, runIdentifier: string): Observable<V2WorkflowRun> {
        return this._http.get<V2WorkflowRun>(`/v2/project/${projKey}/run/${runIdentifier}`);
    }

    restart(projKey: string, runIdentifier: string): Observable<V2WorkflowRun> {
        return this._http.put<V2WorkflowRun>(`/v2/project/${projKey}/run/${runIdentifier}/restart`, null);
    }

    stop(projKey: string, runIdentifier: string) {
        return this._http.post(`/v2/project/${projKey}/run/${runIdentifier}/stop`, null);
    }

    getJobs(r: V2WorkflowRun, attempt: number = null): Observable<Array<V2WorkflowRunJob>> {
        let params = new HttpParams();
        if (attempt) {
            params = params.append('attempt', attempt);
        }
        return this._http.get<Array<V2WorkflowRunJob>>(`/v2/project/${r.project_key}/run/${r.id}/job`, { params });
    }

    getResults(r: V2WorkflowRun, attempt: number = null): Observable<Array<WorkflowRunResult>> {
        let params = new HttpParams();
        if (attempt) {
            params = params.append('attempt', attempt);
        }
        return this._http.get<Array<WorkflowRunResult>>(`/v2/project/${r.project_key}/run/${r.id}/result`, { params });
    }

    getRunInfos(r: V2WorkflowRun): Observable<Array<WorkflowRunInfo>> {
        return this._http.get<Array<WorkflowRunInfo>>(`/v2/project/${r.project_key}/run/${r.id}/infos`);
    }

    getRunJobInfos(r: V2WorkflowRun, jobIdentifier: string): Observable<Array<WorkflowRunInfo>> {
        return this._http.get<Array<WorkflowRunInfo>>(`/v2/project/${r.project_key}/run/${r.id}/job/${jobIdentifier}/infos`);
    }

    getAllLogsLinks(run: V2WorkflowRun, jobIdentifier: string): Observable<CDNLogLinks> {
        return this._http.get<CDNLogLinks>(`/v2/project/${run.project_key}/run/${run.id}/job/${jobIdentifier}/logs/links`);
    }

    triggerJob(run: V2WorkflowRun, jobIdentifier: string, inputs: {}): Observable<V2WorkflowRun> {
        return this._http.put<V2WorkflowRun>(`/v2/project/${run.project_key}/run/${run.id}/job/${jobIdentifier}/run`, inputs);
    }
}
