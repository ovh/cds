
import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { AuditWorkflowTemplate } from '../../model/audit.model';
import {
    WorkflowTemplate,
    WorkflowTemplateApplyResult,
    WorkflowTemplateBulk,
    WorkflowTemplateInstance,
    WorkflowTemplateRequest
} from '../../model/workflow-template.model';
import { Workflow } from '../../model/workflow.model';

@Injectable()
export class WorkflowTemplateService {
    constructor(private _http: HttpClient) { }

    getAll(): Observable<Array<WorkflowTemplate>> {
        return this._http.get<Array<WorkflowTemplate>>('/template');
    }

    get(groupName: string, templateSlug: string): Observable<WorkflowTemplate> {
        return this._http.get<WorkflowTemplate>(`/template/${groupName}/${templateSlug}`);
    }

    add(wt: WorkflowTemplate): Observable<WorkflowTemplate> {
        return this._http.post<WorkflowTemplate>('/template', wt);
    }

    update(old: WorkflowTemplate, wt: WorkflowTemplate): Observable<WorkflowTemplate> {
        return this._http.put<WorkflowTemplate>(`/template/${old.group.name}/${old.slug}`, wt);
    }

    delete(wt: WorkflowTemplate): Observable<any> {
        return this._http.delete<any>(`/template/${wt.group.name}/${wt.slug}`);
    }

    apply(groupName: string, templateSlug: string, req: WorkflowTemplateRequest): Observable<WorkflowTemplateApplyResult> {
        return this._http.post<Array<string>>(`/template/${groupName}/${templateSlug}/apply?import=true`,
            req, { observe: 'response' }).pipe().map(res => {
                let headers: HttpHeaders = res.headers;
                let result = new WorkflowTemplateApplyResult();
                result.workflow_name = headers.get('X-Api-Workflow-Name');
                result.msgs = res.body;
                return result;
            });
    }

    getInstance(projectKey: string, workflowName: string): Observable<WorkflowTemplateInstance> {
        return this._http.get<WorkflowTemplateInstance>(`/project/${projectKey}/workflow/${workflowName}/templateInstance`);
    }

    deleteInstance(wt: WorkflowTemplate, wti: WorkflowTemplateInstance): Observable<any> {
        return this._http.delete<any>(`/template/${wt.group.name}/${wt.slug}/instance/${wti.id}`);
    }

    getAudits(groupName: string, templateSlug: string, version?: number): Observable<Array<AuditWorkflowTemplate>> {
        let params = new HttpParams();
        if (version) {
            params = params.append('sinceVersion', String(version));
        }
        return this._http.get<Array<AuditWorkflowTemplate>>(`/template/${groupName}/${templateSlug}/audit`, { params });
    }

    getUsage(groupName: string, templateSlug: string): Observable<Array<Workflow>> {
        return this._http.get<Array<Workflow>>(`/template/${groupName}/${templateSlug}/usage`);
    }

    getInstances(groupName: string, templateSlug: string): Observable<Array<WorkflowTemplateInstance>> {
        return this._http.get<Array<WorkflowTemplateInstance>>(`/template/${groupName}/${templateSlug}/instance`)
            .map(wtis => wtis.map(wti => new WorkflowTemplateInstance(wti)));
    }

    bulk(groupName: string, templateSlug: string, req: WorkflowTemplateBulk): Observable<WorkflowTemplateBulk> {
        return this._http.post<WorkflowTemplateBulk>(`/template/${groupName}/${templateSlug}/bulk`, req);
    }

    getBulk(groupName: string, templateSlug: string, id: number): Observable<WorkflowTemplateBulk> {
        return this._http.get<WorkflowTemplateBulk>(`/template/${groupName}/${templateSlug}/bulk/${id}`);
    }
}
