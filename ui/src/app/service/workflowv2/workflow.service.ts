import { inject, Injectable } from "@angular/core";
import { HttpClient, HttpHeaders, HttpParams } from "@angular/common/http";
import { Observable } from "rxjs";
import { V2WorkflowRun, V2WorkflowRunJob, V2WorkflowRunTriggerJobsRequest, V2WorkflowRunManualRequest, V2WorkflowRunManualResponse, WorkflowRunInfo, WorkflowRunResult } from "../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";
import { CDNLogLink, CDNLogLinks } from "app/model/cdn.model";

/**
 * Asks a run search for runs without the definition they ran. A list shows numbers, statuses and git
 * information; the definition of the workflow each run ran weighs more than all the rest put together
 * and none of it is displayed there.
 */
export const RUN_SUMMARY_HEADERS = new HttpHeaders({ Accept: 'application/vnd.cds.workflow-run-summary+json' });

/**
 * Where a run pushed by the websocket goes in a list showing one page of a search. The API answers
 * whether the run matches the search, not where it sits in it: sort and paging are the list's
 * business, and it has everything to decide - the sort in effect, what it displays, and the dates
 * the run carries.
 *
 * Returns the index to insert the run at, or -1 when this page does not reach it: under a descending
 * sort a run older than the last one shown is on a later page, and under an ascending sort a run
 * that has just started is at the far end of the result.
 */
export function runInsertIndex(runs: Array<V2WorkflowRun>, run: V2WorkflowRun, sort: string, pageSize: number): number {
	const [field, direction] = (sort ?? '').split(':');
	// Compared as instants: the same one can be written with any offset.
	const at = (r: V2WorkflowRun) => new Date(field === 'last_modified' ? r.last_modified : r.started).getTime();

	const value = at(run);
	for (let i = 0; i < runs.length; i++) {
		if (direction === 'asc' ? value <= at(runs[i]) : value >= at(runs[i])) {
			return i;
		}
	}
	// Past the last run shown, so it only belongs to this page while the page has room for it.
	return runs.length < pageSize ? runs.length : -1;
}

@Injectable()
export class V2WorkflowRunService {
    private _http = inject(HttpClient);

    getRun(projKey: string, workflowRunID: string): Observable<V2WorkflowRun> {
        return this._http.get<V2WorkflowRun>(`/v2/project/${projKey}/run/${workflowRunID}`);
    }

    restart(projKey: string, workflowRunID: string): Observable<V2WorkflowRun> {
        return this._http.post<V2WorkflowRun>(`/v2/project/${projKey}/run/${workflowRunID}/restart`, null);
    }

    start(projKey: string, vcsName: string, repoName: string, workflowName: string, data: V2WorkflowRunManualRequest) {
        let encodedRepo = encodeURIComponent(repoName);
        return this._http.post<V2WorkflowRunManualResponse>(`/v2/project/${projKey}/vcs/${vcsName}/repository/${encodedRepo}/workflow/${workflowName}/run`, data);
    }

    stop(projKey: string, workflowRunID: string) {
        return this._http.post(`/v2/project/${projKey}/run/${workflowRunID}/stop`, null);
    }

    stopJob(projKey: string, workflowRunID: string, jobRunID: string) {
        return this._http.post(`/v2/project/${projKey}/run/${workflowRunID}/job/${jobRunID}/stop`, null);
    }

    // Keyed by ids rather than by the run itself: the jobs, the results and the infos of a run can
    // then be read at the same time as the run, instead of waiting for it.
    getJobs(projKey: string, workflowRunID: string, attempt: number = null): Observable<Array<V2WorkflowRunJob>> {
        let params = new HttpParams();
        if (attempt) {
            params = params.append('attempt', attempt);
        }
        return this._http.get<Array<V2WorkflowRunJob>>(`/v2/project/${projKey}/run/${workflowRunID}/job`, { params });
    }

    getResults(projKey: string, workflowRunID: string, attempt: number = null): Observable<Array<WorkflowRunResult>> {
        let params = new HttpParams();
        if (attempt) {
            params = params.append('attempt', attempt);
        }
        return this._http.get<Array<WorkflowRunResult>>(`/v2/project/${projKey}/run/${workflowRunID}/result`, { params });
    }

    getRunInfos(projKey: string, workflowRunID: string): Observable<Array<WorkflowRunInfo>> {
        return this._http.get<Array<WorkflowRunInfo>>(`/v2/project/${projKey}/run/${workflowRunID}/infos`);
    }

    getRunJobInfos(r: V2WorkflowRun, jobRunID: string): Observable<Array<WorkflowRunInfo>> {
        return this._http.get<Array<WorkflowRunInfo>>(`/v2/project/${r.project_key}/run/${r.id}/job/${jobRunID}/infos`);
    }

    getAllLogsLinks(run: V2WorkflowRun, jobRunID: string): Observable<CDNLogLinks> {
        return this._http.get<CDNLogLinks>(`/v2/project/${run.project_key}/run/${run.id}/job/${jobRunID}/logs/links`);
    }

    getRunJobServiceLogsLink(run: V2WorkflowRun, jobRunID: string, serviceName: string): Observable<CDNLogLink> {
        return this._http.get<CDNLogLink>(`/v2/project/${run.project_key}/run/${run.id}/job/${jobRunID}/service/${serviceName}/link`);
    }

    /**
     * Trigger multiple jobs in a single batch call.
     * Maps to POST /v2/project/{projectKey}/run/{workflowRunID}/job
     * with body { job_inputs: { [jobIdentifier]: { [inputName]: value } } }.
     */
    triggerJobs(projKey: string, workflowRunID: string, request: V2WorkflowRunTriggerJobsRequest): Observable<V2WorkflowRun> {
        return this._http.post<V2WorkflowRun>(`/v2/project/${projKey}/run/${workflowRunID}/job`, request);
    }

    getRetries(projectKey: string, runID: string, runJobID: string): Observable<Array<V2WorkflowRunJob>> {
        return this._http.get<Array<V2WorkflowRunJob>>(`/v2/project/${projectKey}/run/${runID}/job/${runJobID}/retry`);
    }
}
