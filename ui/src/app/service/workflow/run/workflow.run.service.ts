import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Observable';
import {Workflow} from '../../../model/workflow.model';
import {WorkflowNodeRun, WorkflowRun, WorkflowRunRequest} from '../../../model/workflow.run.model';
import {HttpClient} from '@angular/common/http';

@Injectable()
export class WorkflowRunService {

    constructor(private _http: HttpClient) {
    }

    /**
     * Call API to create a run workflow
     * @param key Project unique key
     * @param workflow Workflow to create
     */
    runWorkflow(key: string, workflowName: string, request: WorkflowRunRequest): Observable<WorkflowRun> {
        return this._http.post<WorkflowRun>('/project/' + key + '/workflows/' + workflowName + '/runs', request);
    }

    /**
     * Call API to get history from node run
     * @param {string} key Project unique key
     * @param {string} workflowName Workflow name
     * @param {number} number Workflow Run number
     * @param {number} nodeID Workflow Run node ID
     * @returns {Observable<Array<WorkflowNodeRun>>}
     */
    nodeRunHistory(key: string, workflowName: string, number: number, nodeID: number): Observable<Array<WorkflowNodeRun>> {
        return this._http.get<Array<WorkflowNodeRun>>(
            '/project/' + key + '/workflows/' + workflowName + '/runs/' + number + '/nodes/' + nodeID + '/history');
    }

    /**
     * Get workflow Run
     * @param {string} key Project unique key
     * @param {string} workflowName Workflow name to get
     * @param {number} number Number of the workflow run
     * @returns {Observable<WorkflowRun>}
     */
    getWorkflowRun(key: string, workflowName: string, number: number): Observable<WorkflowRun> {
        return this._http.get<WorkflowRun>('/project/' + key + '/workflows/' + workflowName + '/runs/' + number);
    }

    /**
     * Stop a workflow run
     * @param {string} key Project unique key
     * @param {string} workflowName Workflow name
     * @param {number} number Number of the workflow run
     * @returns {Observable<boolean>}
     */
    stopWorkflowRun(key: string, workflowName: string, num: number): Observable<boolean> {
        return this._http.post('/project/' + key + '/workflows/' + workflowName + '/runs/' + num + '/stop', null).map(() => true);
    }

    /**
     * Stop a workflow node run
     * @param {string} key Project unique key
     * @param {string} workflowName Workflow name
     * @param {number} number Number of the workflow run
     * @param {number} id of the node run to stop
     * @returns {Observable<boolean>}
     */
    stopNodeRun(key: string, workflowName: string, num: number, id: number): Observable<boolean> {
        return this._http.post('/project/' + key + '/workflows/' + workflowName + '/runs/' + num + '/nodes/' + id + '/stop', null)
            .map (() => true);
    }

    /**
     * Get workflow tags
     * @param {string} key Project unique key
     * @param {string} workflowName Workflow name
     * @returns {Observable<{}>}
     */
    getTags(key: string, workflowName: string): Observable<Map<string, Array<string>>> {
        return this._http.get<Map<string, Array<string>>>('/project/' + key + '/workflows/' + workflowName + '/runs/tags');
    }

    /**
     * Resync pipeline inside workflow run
     * @param {string} key Project unique key
     * @param {Workflow} workflow Workflow
     * @param {number} workflow_run_id Workflow run id to resync
     */
    resync(key: string, workflow: Workflow, workflow_run_id: number): Observable<WorkflowRun> {
        return this._http.post<WorkflowRun>(
            '/project/' + key + '/workflows/' + workflow.name + '/runs/' + workflow_run_id + '/resync', null);
    }
}
