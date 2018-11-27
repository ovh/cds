
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import {
    WorkflowTemplate,
    WorkflowTemplateApplyResult,
    WorkflowTemplateInstance,
    WorkflowTemplateRequest
} from '../../model/workflow-template.model';

@Injectable()
export class WorkflowTemplateService {
    constructor(private _http: HttpClient) {
    }

    getWorkflowTemplates(): Observable<Array<WorkflowTemplate>> {
        return this._http.get<Array<WorkflowTemplate>>('/template');
    }

    getWorkflowTemplate(groupName: string, templateSlug: string): Observable<WorkflowTemplate> {
        return this._http.get<WorkflowTemplate>('/template/' + groupName + '/' + templateSlug);
    }

    addWorkflowTemplate(wt: WorkflowTemplate): Observable<WorkflowTemplate> {
        return this._http.post<WorkflowTemplate>('/template', wt);
    }

    updateWorkflowTemplate(old: WorkflowTemplate, wt: WorkflowTemplate): Observable<WorkflowTemplate> {
        return this._http.put<WorkflowTemplate>('/template/' + old.group.name + '/' + old.slug, wt);
    }

    deleteWorkflowTemplate(wt: WorkflowTemplate): Observable<any> {
        return this._http.delete<any>('/template/' + wt.group.name + '/' + wt.slug);
    }

    applyWorkflowTemplate(groupName: string, templateSlug: string, req: WorkflowTemplateRequest): Observable<WorkflowTemplateApplyResult> {
        return this._http.post<Array<string>>('/template/' + groupName + '/' + templateSlug + '/apply?import=true',
            req, { observe: 'response' }).pipe().map(res => {
                let headers: HttpHeaders = res.headers;
                let result = new WorkflowTemplateApplyResult();
                result.workflow_name = headers.get('X-Api-Workflow-Name');
                result.msgs = res.body;
                return result;
            });
    }

    getWorkflowTemplateInstance(projectKey: string, workflowName: string): Observable<WorkflowTemplateInstance> {
        return this._http.get<WorkflowTemplateInstance>('/project/' + projectKey + '/workflow/' + workflowName + '/templateInstance');
    }
}
