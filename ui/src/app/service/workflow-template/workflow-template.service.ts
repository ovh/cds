
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

}
