import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Observable';
import {Workflow, WorkflowTriggerConditionCache} from '../../model/workflow.model';
import {HttpClient, HttpParams, HttpHeaders} from '@angular/common/http';
import {GroupPermission} from '../../model/group.model';
import {deepClone} from 'fast-json-patch/lib/core';

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
     * Get the given workflow from API in export format
     * @param key Project unique key
     * @param workflowName Workflow Name
     */
    getWorkflowExport(key: string, workflowName: string): Observable<string> {
        let params = new HttpParams();
        params = params.append('format', 'yaml');

        return this._http.get<string>('/project/' + key + '/export/workflows/' + workflowName, {params, responseType: <any>'text'});
    }

    /**
     * Get the given workflow from API in export format
     * @param key Project unique key
     * @param workflowName Workflow Name
     */
    previewWorkflowImport(key: string, workflowImportCode: string): Observable<Workflow> {
        let params = new HttpParams();
        params = params.append('format', 'yaml');
        let headers = new HttpHeaders();
        headers = headers.append('Content-Type', 'application/x-yaml');

        return this._http.post<Workflow>('/project/' + key + '/preview/workflows', workflowImportCode, {headers, params});
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
        let w = deepClone(workflow);
        Workflow.prepareRequestForAPI(w);
        return this._http.put<Workflow>('/project/' + key + '/workflows/' + name, w);
    }

    /**
     * Import a workflow
     * @param key Project unique key
     * @param workflow WorkflowCode to import
     */
    importWorkflow(key: string, workflowCode: string): Observable<Workflow> {
        let headers = new HttpHeaders();
        headers = headers.append('Content-Type', 'application/x-yaml');
        let params = new HttpParams();
        params = params.append('force', 'true');

        return this._http.post<Workflow>(`/project/${key}/import/workflows`, workflowCode, {headers, params});
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
