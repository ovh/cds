
import {map} from 'rxjs/operators';
import {Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {HttpClient} from '@angular/common/http';
import {WorkflowHookModel, WorkflowHookTask} from '../../model/workflow.hook.model';
import {Workflow, WorkflowNode} from '../../model/workflow.model';
import {Project} from '../../model/project.model';


@Injectable()
export class HookService {

    constructor(private _http: HttpClient) {
    }

    getHookModel(p: Project, w: Workflow, n: WorkflowNode): Observable<Array<WorkflowHookModel>> {
        return this._http.get<Array<WorkflowHookModel>>('/project/' + p.key + '/workflow/' + w.name +
            '/node/' + n.id + '/hook/model').pipe(map(ms => {
            return ms;
        }));
    }

    getHookLogs(projectKey: string, workflowName: string, uuid: string): Observable<WorkflowHookTask> {
      return this._http.get<WorkflowHookTask>(`/project/${projectKey}/workflows/${workflowName}/hooks/${uuid}`);
    }
}
