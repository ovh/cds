import {Injectable} from '@angular/core';
import {Http} from '@angular/http';
import {Observable} from 'rxjs/Observable';
import {Workflow} from '../../../model/workflow.model';
import {WorkflowRun} from '../../../model/workflow.run.model';

@Injectable()
export class WorkflowRunService {

    constructor(private _http: Http) {
    }

    /**
     * Call API to create a run workflow
     * @param key Project unique key
     * @param workflow Workflow to create
     */
    runWorkflow(key: string, workflow: Workflow, payload: {}): Observable<WorkflowRun> {
        return this._http.post('/project/' + key + '/workflows/' + workflow.name + '/runs', payload).map(res => res.json());
    }
}
