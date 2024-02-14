import { Injectable } from "@angular/core";
import { HttpClient } from "@angular/common/http";
import { Observable } from "rxjs";
import { V2WorkflowRun, V2WorkflowRunJob, WorkflowRunInfo } from "../../model/v2.workflow.run.model";
import { CDNLogLinks } from "../../model/pipeline.model";

@Injectable()
export class V2WorkflowRunService {
    constructor(
        private _http: HttpClient
    ) { }

    getRun(projKey: string, runIdentifier: string): Observable<V2WorkflowRun> {
        return this._http.get<V2WorkflowRun>(`/v2/project/${projKey}/run/${runIdentifier}`);
    }

    getJobs(r: V2WorkflowRun): Observable<Array<V2WorkflowRunJob>> {
        return this._http.get<Array<V2WorkflowRunJob>>(`/v2/project/${r.project_key}/run/${r.id}/job`);
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
