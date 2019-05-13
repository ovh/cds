import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { deepClone } from 'fast-json-patch/lib/core';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';
import { GroupPermission } from '../../model/group.model';
import { Operation } from '../../model/operation.model';
import { Label } from '../../model/project.model';
import { Workflow, WorkflowPull, WorkflowTriggerConditionCache } from '../../model/workflow.model';

@Injectable()
export class WorkflowService {
    constructor(private _http: HttpClient) { }

    /**
     * Get the given workflow from API
     * @param projectKey Project unique key
     * @param workflowName Workflow Name
     */
    getWorkflow(projectKey: string, workflowName: string): Observable<Workflow> {
        let params = new HttpParams();
        params = params.append('withUsage', 'true');
        params = params.append('withAudits', 'true');
        params = params.append('withTemplate', 'true');
        params = params.append('withAsCodeEvents', 'true');
        return this._http.get<Workflow>(`/project/${projectKey}/workflows/${workflowName}`, { params });
    }

    /**
     * Pull the given workflow from API
     * @param projectKey Project unique key
     * @param workflowName Workflow Name
     */
    pullWorkflow(projectKey: string, workflowName: string): Observable<WorkflowPull> {
        let params = new HttpParams();
        params = params.append('json', 'true');
        return this._http.get<WorkflowPull>(`/project/${projectKey}/pull/workflows/${workflowName}`, { params });
    }

    /**
     * Get the given workflow from API in export format
     * @param projectKey Project unique key
     * @param workflowName Workflow Name
     */
    getWorkflowExport(projectKey: string, workflowName: string): Observable<string> {
        let params = new HttpParams();
        params = params.append('format', 'yaml');
        return this._http.get<string>(`/project/${projectKey}/export/workflows/${workflowName}`, { params, responseType: <any>'text' });
    }

    /**
     * Get the given workflow from API in export format
     * @param projectKey Project unique key
     * @param workflowName Workflow Name
     */
    previewWorkflowImport(projectKey: string, workflowImportCode: string): Observable<Workflow> {
        let params = new HttpParams();
        params = params.append('format', 'yaml');
        let headers = new HttpHeaders();
        headers = headers.append('Content-Type', 'application/x-yaml');

        return this._http.post<Workflow>(`/project/${projectKey}/preview/workflows`, workflowImportCode, { headers, params });
    }

    /**
     * Call API to create a new workflow
     * @param projectKey Project unique key
     * @param workflow Workflow to create
     */
    addWorkflow(projectKey: string, workflow: Workflow): Observable<Workflow> {
        return this._http.post<Workflow>(`/project/${projectKey}/workflows`, workflow);
    }

    /**
     * Update a workflow
     * @param projectKey Project unique key
     * @param workflow Workflow to update
     */
    updateWorkflow(projectKey: string, name: string, workflow: Workflow): Observable<Workflow> {
        let w = deepClone(workflow);
        Workflow.prepareRequestForAPI(w);
        return this._http.put<Workflow>(`/project/${projectKey}/workflows/${name}`, w);
    }

    /**
     * Update a workflow favorite
     * @param projectKey Project unique key
     * @param workflow Workflow to update
     */
    updateFavorite(projectKey: string, name: string): Observable<Workflow> {
        return this._http.post<Workflow>('/user/favorite', {
            type: 'workflow',
            project_key: projectKey,
            workflow_name: name,
        });
    }

    /**
     * Import a workflow
     * @param projectKey Project unique key
     * @param workflow WorkflowCode to import
     */
    importWorkflow(projectKey: string, workflowName: string, workflowCode: string): Observable<Workflow> {
        let headers = new HttpHeaders();
        headers = headers.append('Content-Type', 'application/x-yaml');

        if (workflowName) {
            return this._http.put<Workflow>(`/project/${projectKey}/import/workflows/${workflowName}`, workflowCode, { headers });
        }

        return this._http.post<Workflow>(`/project/${projectKey}/import/workflows`, workflowCode, { headers });
    }

    /**
     * Rollback a workflow
     * @param projectKey Project unique key
     * @param workflow WorkflowCode to import
     * @param auditId audit id to rollback
     */
    rollbackWorkflow(projectKey: string, workflowName: string, auditId: number): Observable<Workflow> {
        return this._http.post<Workflow>(`/project/${projectKey}/workflows/${workflowName}/rollback/${auditId}`, {});
    }

    /**
     * Delete workflow
     * @param projectKey Project unique key
     * @param workflow Workflow to delete
     * @returns {Observable<boolean>}
     */
    deleteWorkflow(projectKey: string, workflow: Workflow): Observable<boolean> {
        return this._http.delete(`/project/${projectKey}/workflows/${workflow.name}`).pipe(map(res => true));
    }

    /**
     * Get workflow trigger condition
     * @param projectKey Project unique key
     * @param workflow Workflow to delete
     * @returns {Observable<boolean>}
     */
    getTriggerCondition(projectKey: string, workflowName: string, nodeID: number): Observable<WorkflowTriggerConditionCache> {
        return this._http.get<WorkflowTriggerConditionCache>(
            `/project/${projectKey}/workflows/${workflowName}/node/${nodeID}/triggers/condition`);
    }

    /**
     * Get workflow join trigger condition
     * @param projectKey Project unique key
     * @param workflow Workflow to delete
     * @returns {Observable<boolean>}
     */
    getTriggerJoinCondition(projectKey: string, workflowName: string, joinID: number): any {
        return this._http.get(`/project/${projectKey}/workflows/${workflowName}/join/${joinID}/triggers/condition`);
    }

    /**
     * Add a permission on a workflow
     * @param {string} projectKey Project key
     * @param {string} workflowName Workflow name
     * @param {GroupPermission} gp Permission to add
     * @returns {Observable<Workflow>}
     */
    addPermission(projectKey: string, workflowName: string, gp: GroupPermission): Observable<Workflow> {
        return this._http.post<Workflow>(`/project/${projectKey}/workflows/${workflowName}/groups`, gp);
    }

    /**
     * Update a permission on a workflow
     * @param {string} projectKey Project unique key
     * @param {string} workflowName Workflow name
     * @param {GroupPermission} gp Permission to update
     * @returns {Observable<Workflow>}
     */
    updatePermission(projectKey: string, workflowName: string, gp: GroupPermission): Observable<Workflow> {
        return this._http.put<Workflow>(`/project/${projectKey}/workflows/${workflowName}/groups/${gp.group.name}`, gp);
    }

    /**
     * Delete Permission
     * @param {string} projectKey Project unique key
     * @param {string} workflowName Workflow Name
     * @param {GroupPermission} gp Permission to delete
     * @returns {Observable<Workflow>}
     */
    deletePermission(projectKey: string, workflowName: string, gp: GroupPermission): Observable<Workflow> {
        return this._http.delete<Workflow>(`/project/${projectKey}/workflows/${workflowName}/groups/${gp.group.name}`);
    }

    /**
     * Link a label on a workflow
     * @param {string} projectKey Project unique key
     * @param {string} workflowName Workflow Name
     * @param {Label} label label to link
     * @returns {Observable<Label>}
     */
    linkLabel(projectKey: string, workflowName: string, label: Label): Observable<Label> {
        return this._http.post<Label>(`/project/${projectKey}/workflows/${workflowName}/label`, label);
    }

    /**
     * Link a label on a workflow
     * @param {string} projectKey Project unique key
     * @param {string} workflowName Workflow Name
     * @param {labelId} labelId labelId to unlink
     * @returns {Observable<Label>}
     */
    unlinkLabel(projectKey: string, workflowName: string, labelId: number): Observable<null> {
        return this._http.delete<null>(`/project/${projectKey}/workflows/${workflowName}/label/${labelId}`);
    }

    /**
     * Transform the workflow as  workflow as code
     * @param projectKey
     * @param workflowName
     */
    migrateAsCode(projectKey: string, workflowName: string): Observable<Operation> {
        return this._http.post<Operation>(`/project/${projectKey}/workflows/${workflowName}/ascode`, null);
    }

    /**
     * Resync As Code PR
     * @param projectKey
     * @param workflowName
     */
    resyncPRAsCode(projectKey: string, workflowName: string) {
        return this._http.post(`/project/${projectKey}/workflows/${workflowName}/ascode/resync/pr`, null)
    }
}
