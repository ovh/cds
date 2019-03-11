import { Injectable } from '@angular/core';
import { List, Map } from 'immutable';
import { BehaviorSubject, Observable } from 'rxjs';
import { NavbarRecentData } from '../../model/navbar.model';
import { Operation } from '../../model/operation.model';
import { Workflow } from '../../model/workflow.model';
import { WorkflowService } from './workflow.service';

@Injectable()
export class WorkflowStore {

    static RECENT_WORKFLOWS_KEY = 'CDS-RECENT-WORKFLOWS';
    WORKFLOW_ORIENTATION_KEY = 'CDS-WORKFLOW-ORIENTATION';

    // List of all workflows.
    private _workflows: BehaviorSubject<Map<string, Workflow>> = new BehaviorSubject(Map<string, Workflow>());

    private _recentWorkflows: BehaviorSubject<List<NavbarRecentData>> = new BehaviorSubject(List<NavbarRecentData>());


    constructor(
        private _workflowService: WorkflowService
    ) {
        this.loadRecentWorkflows();
    }

    loadRecentWorkflows(): void {
        let arrayWorkflows = JSON.parse(localStorage.getItem(WorkflowStore.RECENT_WORKFLOWS_KEY));
        this._recentWorkflows.next(List.of(...arrayWorkflows));
    }

    /**
     * Get recent workflow.
     * @returns {Observable<List<Workflow>>}
     */
    getRecentWorkflows(): Observable<List<NavbarRecentData>> {
        return new Observable<List<NavbarRecentData>>(fn => this._recentWorkflows.subscribe(fn));
    }

    /**
     * Update recent workflow viewed.
     * @param key Project unique key
     * @param workflow Workflow to add
     */
    updateRecentWorkflow(key: string, workflow: Workflow): void {
        let navbarRecentData = new NavbarRecentData();
        navbarRecentData.project_key = key;
        navbarRecentData.name = workflow.name;
        let currentRecentWorkflows: Array<NavbarRecentData> = JSON.parse(localStorage.getItem(WorkflowStore.RECENT_WORKFLOWS_KEY));
        if (currentRecentWorkflows) {
            let index: number = currentRecentWorkflows.findIndex(w =>
                w.name === navbarRecentData.name && w.project_key === navbarRecentData.project_key
            );
            if (index >= 0) {
                currentRecentWorkflows.splice(index, 1);
            }
        } else {
            currentRecentWorkflows = new Array<NavbarRecentData>();
        }
        currentRecentWorkflows.splice(0, 0, navbarRecentData);
        currentRecentWorkflows = currentRecentWorkflows.splice(0, 15);
        localStorage.setItem(WorkflowStore.RECENT_WORKFLOWS_KEY, JSON.stringify(currentRecentWorkflows));
        this._recentWorkflows.next(List(currentRecentWorkflows));
    }

    getDirection(key: string, name: string) {
        let o = localStorage.getItem(this.WORKFLOW_ORIENTATION_KEY);
        if (o) {
            let j = JSON.parse(o);
            if (j[key + '-' + name]) {
                return j[key + '-' + name];
            }
        }
        return 'LR';
    }

    setDirection(key: string, name: string, o: string) {
        let ls = localStorage.getItem(this.WORKFLOW_ORIENTATION_KEY);
        let j = {};
        if (ls) {
            j = JSON.parse(ls);
        }
        j[key + '-' + name] = o;
        localStorage.setItem(this.WORKFLOW_ORIENTATION_KEY, JSON.stringify(j));
    }

    externalModification(wfKey: string) {
        let cache = this._workflows.getValue();
        let wfToUpdate = cache.get(wfKey);
        if (wfToUpdate) {
            wfToUpdate.externalChange = true;
            this._workflows.next(cache.set(wfKey, wfToUpdate));
        }
    }

    migrateAsCode(key: string, workflowName: string): Observable<Operation> {
        return this._workflowService.migrateAsCode(key, workflowName);
    }
}
