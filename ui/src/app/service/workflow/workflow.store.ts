import {ProjectStore} from '../project/project.store';
import {BehaviorSubject} from 'rxjs/BehaviorSubject';
import {Workflow} from '../../model/workflow.model';
import {Injectable} from '@angular/core';
import {List, Map} from 'immutable';
import {Observable} from 'rxjs/Observable';
import {WorkflowService} from './workflow.service';

@Injectable()
export class WorkflowStore {

    static RECENT_WORKFLOW_KEY = 'CDS-RECENT-WORKFLOW';

    // List of all workflows.
    private _workflows: BehaviorSubject<Map<string, Workflow>> = new BehaviorSubject(Map<string, Workflow>());

    private _recentWorkflows: BehaviorSubject<List<Workflow>> = new BehaviorSubject(List<Workflow>());


    constructor(private _projectStore: ProjectStore, private _workflowService: WorkflowService) {
        this.loadRecentWorkflows();
    }

    loadRecentWorkflows(): void {
        let arrayWorkflows = JSON.parse(localStorage.getItem(WorkflowStore.RECENT_WORKFLOW_KEY));
        this._recentWorkflows.next(List.of(...arrayWorkflows));
    }

    /**
     * Get recent workflow.
     * @returns {Observable<List<Workflow>>}
     */
    getRecentWorkflows(): Observable<List<Workflow>> {
        return new Observable<List<Workflow>>(fn => this._recentWorkflows.subscribe(fn));
    }

    /**
     * Use by router to preload workflow
     * @param key
     * @param workflowName Workflow name
     * @returns {Observable<Workflow>}
     */
    getWorkflowResolver(key: string, workflowName: string): Observable<Workflow> {
        let store = this._workflows.getValue();
        let workflowKey = key + '-' + workflowName;
        if (store.size === 0 || !store.get(workflowKey)) {
            return this._workflowService.getWorkflow(key, workflowName).map( res => {
                this._workflows.next(store.set(workflowKey, res));
                return res;
            });
        } else {
            return Observable.of(store.get(workflowKey));
        }
    }

    /**
     * Get workflows
     * @returns {Observable<Application>}
     */
    getWorkflows(key: string, workflowName?: string): Observable<Map<string, Workflow>> {
        let store = this._workflows.getValue();
        let workflowKey = key + '-' + workflowName;
        if (workflowName && !store.get(workflowKey)) {
            this.resync(key, workflowName);
        }
        return new Observable<Map<string, Workflow>>(fn => this._workflows.subscribe(fn));
    }

    resync(key: string, workflowName: string) {
        let store = this._workflows.getValue();
        let workflowKey = key + '-' + workflowName;
        this._workflowService.getWorkflow(key, workflowName).subscribe(res => {
            this._workflows.next(store.set(workflowKey, res));
        }, err => {
            this._workflows.error(err);
        });
    }

    /**
     * Add a new workflow in a project
     * @param key Project unique key
     * @param workflow Workflow to add
     */
    addWorkflow(key: string, workflow: Workflow): Observable<Workflow> {
        return this._workflowService.addWorkflow(key, workflow);
    }
}
