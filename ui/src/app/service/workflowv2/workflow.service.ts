import {Injectable} from "@angular/core";
import {HttpClient} from "@angular/common/http";
import {Observable} from "rxjs";
import {V2WorkflowRun, V2WorkflowRunJob} from "../../model/v2.workflow.run.model";

@Injectable()
export class V2WorkflowRunService {

    constructor(private _http: HttpClient) {
    }

    listRun(projKey: string, vcs: string, repo: string, workflow: string, branch: string): Observable<Array<V2WorkflowRun>> {
        let repoEncoded = encodeURIComponent(repo);
        return this._http.get<Array<V2WorkflowRun>>(`/v2/project/${projKey}/vcs/${vcs}/repository/${repoEncoded}/workflow/${workflow}/run`)
    }

    getJobs(r: V2WorkflowRun): Observable<Array<V2WorkflowRunJob>> {
        return this._http.get<Array<V2WorkflowRunJob>>(`/v2/project/${r.project_key}/vcs/${r.vcs_server_id}/repository/${r.repository_id}/workflow/${r.workflow_name}/run/${r.run_number}/jobs`)
    }
}
