import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { Project } from '../../model/project.model';
import { WorkflowHookModel, WorkflowHookTask } from '../../model/workflow.hook.model';
import { Workflow, WorkflowNode } from '../../model/workflow.model';

@Injectable()
export class HookService {

    constructor(private _http: HttpClient) {
    }

    getHookModel(p: Project, w: Workflow, n: WorkflowNode): Observable<Array<WorkflowHookModel>> {
        return this._http.get<Array<WorkflowHookModel>>(
            '/project/' + p.key + '/workflow/' + w.name + '/node/' + n.id + '/hook/model');
    }

    getOutgoingHookModel(p: Project, w: Workflow, n: WorkflowNode): Observable<Array<WorkflowHookModel>> {
        return this._http.get<Array<WorkflowHookModel>>(
            '/project/' + p.key + '/workflow/' + w.name + '/node/' + n.id + '/outgoinghook/model'
        );
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
