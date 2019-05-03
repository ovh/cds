import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { Operation } from '../../model/operation.model';
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
     * Transform the workflow as  workflow as code
     * @param projectKey
     * @param workflowName
     */
    migrateAsCode(projectKey: string, workflowName: string): Observable<Operation> {
        return this._http.post<Operation>(`/project/${projectKey}/workflows/${workflowName}/ascode`, null);
    }
}
