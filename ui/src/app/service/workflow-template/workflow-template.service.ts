
import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { WorkflowTemplate } from '../../model/workflow-template.model';

/**
 * Service to get workflow template
 */
@Injectable()
export class WorkflowTemplateService {

    constructor(private _http: HttpClient) {
    }

    /**
     * Get the list of available workflow templates
     * @returns {Observable<WorkflowTemplate[]>}
     */
    getWorkflowTemplates(): Observable<Array<WorkflowTemplate>> {
        return this._http.get<Array<WorkflowTemplate>>('/template');
    }

    /**
     * Get a workflow template by id
     * @returns {Observable<WorkflowTemplate>}
     */
    getWorkflowTemplate(groupName: string, templateSlug: string): Observable<WorkflowTemplate> {
        return this._http.get<WorkflowTemplate>('/template/' + groupName + '/' + templateSlug);
    }

    /**
     * Add a workflow template
     * @returns {Observable<WorkflowTemplate>}
     */
    addWorkflowTemplate(wt: WorkflowTemplate): Observable<WorkflowTemplate> {
        return this._http.post<WorkflowTemplate>('/template', wt);
    }

    /**
     * Update a workflow template
     * @returns {Observable<WorkflowTemplate>}
     */
    updateWorkflowTemplate(old: WorkflowTemplate, wt: WorkflowTemplate): Observable<WorkflowTemplate> {
        return this._http.put<WorkflowTemplate>('/template/' + old.group.name + '/' + old.slug, wt);
    }

    /**
     * Delete a workflow template
     * @returns {Observable<any>}
     */
    deleteWorkflowTemplate(wt: WorkflowTemplate): Observable<any> {
        return this._http.delete<any>('/template/' + wt.group.name + '/' + wt.slug);
    }

}
