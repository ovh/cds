import {HttpClient, HttpParams} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Commit} from 'app/model/repositories.model';
import {Workflow} from 'app/model/workflow.model';
import {
    RunNumber,
    WorkflowNodeRun,
    WorkflowRun,
    WorkflowRunRequest,
    WorkflowRunResult,
    WorkflowRunSummary
} from 'app/model/workflow.run.model';
import {Observable} from 'rxjs';
import {map} from 'rxjs/operators';

@Injectable()
export class WorkflowRunService {

    constructor(private _http: HttpClient) {
    }

    /**
     * List workflow runs for the given workflow
     */
    runs(key: string, workflowName: string, limit: string, offset?: string, filters?: {}): Observable<Array<WorkflowRunSummary>> {
        let url = '/project/' + key + '/workflows/' + workflowName + '/runs';
        let params = new HttpParams();
        params = params.append('limit', limit);
        if (offset) {
            params = params.append('offset', offset);
        }
        if (filters) {
            Object.keys(filters).forEach((tag) => params = params.append(tag, filters[tag]));
        }

        return this._http.get<Array<WorkflowRunSummary>>(url, {params});
    }

    /**
     * Call API to create a run workflow
     *
     * @param key Project unique key
     * @param workflow Workflow to create
     */
    runWorkflow(key: string, workflowName: string, request: WorkflowRunRequest): Observable<WorkflowRun> {
        return this._http.post<WorkflowRun>('/project/' + key + '/workflows/' + workflowName + '/runs', request);
    }

    /**
     * Call API to get history from node run
     *
     * @param key Project unique key
     * @param workflowName Workflow name
     * @param number Workflow Run number
     * @param nodeID Workflow Run node ID
     * @returns
     */
    nodeRunHistory(key: string, workflowName: string, number: number, nodeID: number): Observable<Array<WorkflowNodeRun>> {
        return this._http.get<Array<WorkflowNodeRun>>(
            '/project/' + key + '/workflows/' + workflowName + '/runs/' + number + '/nodes/' + nodeID + '/history');
    }

    /**
     * Get workflow Run
     *
     * @param key Project unique key
     * @param workflowName Workflow name to get
     * @param number Number of the workflow run
     * @returns
     */
    getWorkflowRun(key: string, workflowName: string, number: number): Observable<WorkflowRun> {
        return this._http.get<WorkflowRun>('/project/' + key + '/workflows/' + workflowName + '/runs/' + number).pipe(map(wr => wr));
    }

    /**
     * Get workflow node run
     *
     * @param key Project unique key
     * @param workflowName Workflow name
     * @param number Run number
     * @param nodeRunID Node run Identifier
     * @returns
     */
    getWorkflowNodeRun(key: string, workflowName: string, number: number, nodeRunID): Observable<WorkflowNodeRun> {
        return this._http.get<WorkflowNodeRun>('/project/' + key + '/workflows/' + workflowName +
            '/runs/' + number + '/nodes/' + nodeRunID);
    }

    /**
     * Get all result for the given workflow node run
     *
     * @param key Project unique key
     * @param workflowName Workflow name
     * @param number Run number
     * @param nodeRunID Node run Identifier
     */
    getWorkflowNodeRunResults(key: string, workflowName: string, number: number, nodeRunID: number): Observable<Array<WorkflowRunResult>> {
        return this._http.get<Array<WorkflowRunResult>>(
            `/project/${key}/workflows/${workflowName}/runs/${number}/nodes/${nodeRunID}/results`);
    }

    /**
     * Stop a workflow run
     *
     * @param key Project unique key
     * @param workflowName Workflow name
     * @param number Number of the workflow run
     * @returns
     */
    stopWorkflowRun(key: string, workflowName: string, num: number): Observable<boolean> {
        return this._http.post('/project/' + key + '/workflows/' + workflowName + '/runs/' + num + '/stop', null).pipe(map(() => true));
    }

    /**
     * Stop a workflow node run
     *
     * @param key Project unique key
     * @param workflowName Workflow name
     * @param number Number of the workflow run
     * @param id of the node run to stop
     * @returns
     */
    stopNodeRun(key: string, workflowName: string, num: number, id: number): Observable<boolean> {
        return this._http.post('/project/' + key + '/workflows/' + workflowName + '/runs/' + num + '/nodes/' + id + '/stop', null).pipe(
            map (() => true));
    }

    /**
     * Get workflow tags
     *
     * @param key Project unique key
     * @param workflowName Workflow name
     * @returns
     */
    getTags(key: string, workflowName: string): Observable<Map<string, Array<string>>> {
        return this._http.get<Map<string, Array<string>>>('/project/' + key + '/workflows/' + workflowName + '/runs/tags');
    }

    /**
     * Resync workflow run vcs status
     *
     * @param key Project unique key
     * @param workflow Workflow
     * @param workflowNum Workflow run id to resync
     */
    resyncVCSStatus(key: string, workflowName: string, workflowNum: number): Observable<WorkflowRun> {
        return this._http.post<WorkflowRun>(
            '/project/' + key + '/workflows/' + workflowName + '/runs/' + workflowNum + '/vcs/resync', {});
    }

    /**
     * Get commits linked to a workflow run
     *
     * @param key Project unique key
     * @param workflowName Workflow name
     * @param workflowNumber Workflow number
     * @param workflowNodeId Workflow node id
     */
    getCommits(key: string, workflowName: string, workflowNumber: number,
        workflowNodeName: string, branch?: string, hash?: string, remote?: string): Observable<Commit[]> {

        let params = new HttpParams();
        if (branch) {
            params = params.append('branch', branch);
        }
        if (hash) {
          params = params.append('hash', hash);
        }
        if (remote) {
          params = params.append('remote', remote);
        }
        return this._http.get<Commit[]>(
            `/project/${key}/workflows/${workflowName}/runs/${workflowNumber}/${workflowNodeName}/commits`, {params});
    }

    /**
     * Get current run number for the given workflow
     *
     * @param key Project unique key
     * @param workflow Workflow
     * @returns
     */
    getRunNumber(key: string, workflow: Workflow): Observable<RunNumber> {
        return this._http.get<RunNumber>('/project/' + key + '/workflows/' + workflow.name + '/runs/num');
    }

    /**
     * Update run number
     *
     * @param key Project unique key
     * @param workflow Workflow to update
     * @param num New run number
     * @returns
     */
    updateRunNumber(key: string, workflow: Workflow, num: number): Observable<boolean> {
        let r = new RunNumber();
        r.num = num;
        return this._http.post<void>('/project/' + key + '/workflows/' + workflow.name + '/runs/num', r).pipe(map(() => true));
    }
}
