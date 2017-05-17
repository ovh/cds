import {Injectable} from '@angular/core';
import {Http} from '@angular/http';
import {Observable} from 'rxjs/Observable';
import {Workflow} from '../../model/workflow.model';

@Injectable()
export class WorkflowService {

    constructor(private _http: Http) {
    }

    /**
     * Get the given workflow from API
     * @param key Project unique key
     * @param workflowName Workflow Name
     */
    getWorkflow(key: string, workflowName: string): Observable<Workflow> {
        return this._http.get('/project/' + key + '/workflows/' + workflowName).map(res => res.json());
    }

    /**
     * Call API to create a new workflow
     * @param key Project unique key
     * @param workflow Workflow to create
     */
    addWorkflow(key: string, workflow: Workflow): Observable<Workflow> {
        return this._http.post('/project/' + key + '/workflows', workflow).map(res => res.json());
    }
}
