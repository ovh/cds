import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Observable';
import {Workflow, WorkflowTriggerConditionCache, WorkflowNode, WorkflowNodeJoin} from '../../model/workflow.model';
import {HttpClient, HttpParams} from '@angular/common/http';
import {GroupPermission} from '../../model/group.model';

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
        let params = new HttpParams();
        params = params.append('withUsage', 'true');

        return this._http.get<Workflow>('/project/' + key + '/workflows/' + workflowName, {params});
    }

    /**
     * Call API to create a new workflow
     * @param key Project unique key
     * @param workflow Workflow to create
     */
    addWorkflow(key: string, workflow: Workflow): Observable<Workflow> {
        return this._http.post<Workflow>('/project/' + key + '/workflows', workflow);
    }

    /**
     * Update a workflow
     * @param key Project unique key
     * @param workflow Workflow to update
     */
    updateWorkflow(key: string, name: string, workflow: Workflow): Observable<Workflow> {
        // reinit node id
        Workflow.reinitID(workflow);
        return this._http.put<Workflow>('/project/' + key + '/workflows/' + name, workflow);
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
        return this._http.get<WorkflowTriggerConditionCache>(
            '/project/' + key + '/workflows/' + workflowName + '/node/' + nodeID + '/triggers/condition');
    }

    getTriggerJoinCondition(key: string, workflowName: string, joinID: number): any {
        return this._http.get('/project/' + key + '/workflows/' + workflowName + '/join/' + joinID + '/triggers/condition')
            ;
    }

    /**
     * Add a permission on a workflow
     * @param {string} key Project key
     * @param {string} workflowName Workflow name
     * @param {GroupPermission} gp Permission to add
     * @returns {Observable<Workflow>}
     */
    addPermission(key: string, workflowName: string, gp: GroupPermission): Observable<Workflow> {
        return this._http.post<Workflow>('/project/' + key + '/workflows/' + workflowName + '/groups', gp);
    }

    /**
     * Update a permission on a workflow
     * @param {string} key Project unique key
     * @param {string} workflowName Workflow name
     * @param {GroupPermission} gp Permission to update
     * @returns {Observable<Workflow>}
     */
    updatePermission(key: string, workflowName: string, gp: GroupPermission): Observable<Workflow> {
        return this._http.put<Workflow>('/project/' + key + '/workflows/' + workflowName + '/groups/' + gp.group.name, gp);
    }

    /**
     * Delete Permission
     * @param {string} key Project unique key
     * @param {string} workflowName Workflow Name
     * @param {GroupPermission} gp Permission to delete
     * @returns {Observable<Workflow>}
     */
    deletePermission(key: string, workflowName: string, gp: GroupPermission): Observable<Workflow> {
        return this._http.delete<Workflow>('/project/' + key + '/workflows/' + workflowName + '/groups/' + gp.group.name);
    }
}
