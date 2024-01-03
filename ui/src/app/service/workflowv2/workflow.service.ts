import {Injectable} from "@angular/core";
import {HttpClient} from "@angular/common/http";
import {Observable} from "rxjs";
import {V2WorkflowRun, V2WorkflowRunJob, WorkflowRunInfo} from "../../model/v2.workflow.run.model";
import {CDNLogLinks} from "../../model/pipeline.model";

@Injectable()
export class V2WorkflowRunService {

    constructor(private _http: HttpClient) {
    }

    listRun(projKey: string, vcs: string, repo: string, workflow: string, branch: string): Observable<Array<V2WorkflowRun>> {
        let repoEncoded = encodeURIComponent(repo);
        return this._http.get<Array<V2WorkflowRun>>(`/v2/project/${projKey}/vcs/${vcs}/repository/${repoEncoded}/workflow/${workflow}/run`);
    }

    getJobs(r: V2WorkflowRun): Observable<Array<V2WorkflowRunJob>> {
        return this._http.get<Array<V2WorkflowRunJob>>(`/v2/project/${r.project_key}/vcs/${r.vcs_server_id}/repository/${r.repository_id}/workflow/${r.workflow_name}/run/${r.run_number}/jobs`);
    }

    getRunInfos(r: V2WorkflowRun): Observable<Array<WorkflowRunInfo>> {
        return this._http.get<Array<WorkflowRunInfo>>(`/v2/project/${r.project_key}/vcs/${r.vcs_server_id}/repository/${r.repository_id}/workflow/${r.workflow_name}/run/${r.run_number}/infos`);
    }

    getRunJob(r: V2WorkflowRun, jobName: string): Observable<V2WorkflowRunJob> {
        return this._http.get<V2WorkflowRunJob>(`/v2/project/${r.project_key}/vcs/${r.vcs_server_id}/repository/${r.repository_id}/workflow/${r.workflow_name}/run/${r.run_number}/jobs/${jobName}`);
    }

    getRunJobInfos(r: V2WorkflowRun, jobIdentifier: string): Observable<Array<WorkflowRunInfo>> {
        return this._http.get<Array<WorkflowRunInfo>>(`/v2/project/${r.project_key}/vcs/${r.vcs_server_id}/repository/${r.repository_id}/workflow/${r.workflow_name}/run/${r.run_number}/jobs/${jobIdentifier}/infos`);
    }

    getAllLogsLinks(run: V2WorkflowRun, jobIdentifier: string): Observable<CDNLogLinks> {
        return this._http.get<CDNLogLinks>(`/v2/project/${run.project_key}/vcs/${run.vcs_server_id}/repository/${run.repository_id}/workflow/${run.workflow_name}/run/${run.run_number}/jobs/${jobIdentifier}/logs/links`);
    }

    triggerJob(run: V2WorkflowRun, jobIdentifier: string, inputs: {}): Observable<V2WorkflowRun> {
        return this._http.put<V2WorkflowRun>(`/v2/project/${run.project_key}/vcs/${run.vcs_server_id}/repository/${run.repository_id}/workflow/${run.workflow_name}/run/${run.run_number}/jobs/${jobIdentifier}/run`, inputs);
    }
}
