import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Observable';
import {Workflow, WorkflowTriggerConditionCache} from '../../model/workflow.model';
import {HttpClient} from '@angular/common/http';

@Injectable()
export class WorkflowService {

    constructor(private _http: HttpClient) {
    }

    /**
     * Get the given workflow from API
     * @param key Project unique key
     * @param workflowName Workflow Name
     */
    getWorkflow(key: string, workflowName: string): Observable<Workflow> {
        return this._http.get('/project/' + key + '/workflows/' + workflowName);
    }

    /**
     * Call API to create a new workflow
     * @param key Project unique key
     * @param workflow Workflow to create
     */
    addWorkflow(key: string, workflow: Workflow): Observable<Workflow> {
        return this._http.post('/project/' + key + '/workflows', workflow);
    }

    /**
     * Update a workflow
     * @param key Project unique key
     * @param workflow Workflow to update
     */
    updateWorkflow(key: string, workflow: Workflow): Observable<Workflow> {
        return this._http.put('/project/' + key + '/workflows/' + workflow.name, workflow);
    }

    /**
     * Delete workflow
     * @param key Project unique key
     * @param workflow Workflow to delete
     * @returns {Observable<boolean>}
     */
    deleteWorkflow(key: string, workflow: Workflow): Observable<boolean> {
        return this._http.delete('/project/' + key + '/workflows/' + workflow.name).map(res => true);
    }

    getTriggerCondition(key: string, workflowName: string, nodeID: number): Observable<WorkflowTriggerConditionCache> {
        return this._http.get('/project/' + key + '/workflows/' + workflowName + '/node/' + nodeID + '/triggers/condition')
            ;
    }

    getTriggerJoinCondition(key: string, workflowName: string, joinID: number): any {
        return this._http.get('/project/' + key + '/workflows/' + workflowName + '/join/' + joinID + '/triggers/condition')
            ;
    }
}
