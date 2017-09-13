import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Observable';
import {Workflow} from '../../../model/workflow.model';
import {WorkflowNodeRun, WorkflowRun} from '../../../model/workflow.run.model';
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
    runWorkflow(key: string, workflow: Workflow, payload: {}): Observable<WorkflowRun> {
        return this._http.post('/project/' + key + '/workflows/' + workflow.name + '/runs', payload);
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
        return this._http.get('/project/' + key + '/workflows/' + workflowName + '/runs/' + number + '/nodes/' + nodeID + '/history');
    }
}
