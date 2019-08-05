import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Project } from 'app/model/project.model';
import { WorkflowHookModel, WorkflowHookTask } from 'app/model/workflow.hook.model';
import {WNode, Workflow} from 'app/model/workflow.model';
import { Observable } from 'rxjs';

@Injectable()
export class HookService {

    constructor(private _http: HttpClient) {
    }

    getHookModel(p: Project, w: Workflow, n: WNode): Observable<Array<WorkflowHookModel>> {
        return this._http.get<Array<WorkflowHookModel>>(
            '/project/' + p.key + '/workflow/' + w.name + '/node/' + n.id + '/hook/model');
    }

    getOutgoingHookModel(): Observable<Array<WorkflowHookModel>> {
        return this._http.get<Array<WorkflowHookModel>>('/workflow/outgoinghook/model');
    }

    getHookLogs(projectKey: string, workflowName: string, uuid: string): Observable<WorkflowHookTask> {
        return this._http.get<WorkflowHookTask>(`/project/${projectKey}/workflows/${workflowName}/hooks/${uuid}`);
    }

    getAdminTasks(sort: string): Observable<Array<WorkflowHookTask>> {
        return this.callServiceHooks<Array<WorkflowHookTask>>('/task' + (sort ? '?sort=' + sort : ''));
    }

    getAdminTaskExecution(uuid: string): Observable<WorkflowHookTask> {
        return this.callServiceHooks<WorkflowHookTask>('/task/' + uuid + '/execution');
    }

    callServiceHooks<T>(query: string): Observable<T> {
        return this._http.get<T>('/admin/services/call?type=hooks&query=' + encodeURIComponent(query));
    }
}
