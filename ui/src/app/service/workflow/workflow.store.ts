import { Injectable } from '@angular/core';
import { NavbarRecentData } from 'app/model/navbar.model';
import { Workflow } from 'app/model/workflow.model';
import { List } from 'immutable';
import { BehaviorSubject, Observable } from 'rxjs';

@Injectable()
export class WorkflowStore {

    static RECENT_WORKFLOWS_KEY = 'CDS-RECENT-WORKFLOWS';
    WORKFLOW_ORIENTATION_KEY = 'CDS-WORKFLOW-ORIENTATION';

    private _recentWorkflows: BehaviorSubject<List<NavbarRecentData>> = new BehaviorSubject(List<NavbarRecentData>());


    constructor(
    ) {
        this.loadRecentWorkflows();
    }

    loadRecentWorkflows(): void {
        let arrayWorkflows = JSON.parse(localStorage.getItem(WorkflowStore.RECENT_WORKFLOWS_KEY));
        this._recentWorkflows.next(List.of(...arrayWorkflows));
    }

    /**
     * Get recent workflow.
     *
     * @returns
     */
    getRecentWorkflows(): Observable<List<NavbarRecentData>> {
        return new Observable<List<NavbarRecentData>>(fn => this._recentWorkflows.subscribe(fn));
    }

    /**
     * Update recent workflow viewed.
     *
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
        if (!key || !name) {
            return;
        }
        let ls = localStorage.getItem(this.WORKFLOW_ORIENTATION_KEY);
        let j = {};
        if (ls) {
            j = JSON.parse(ls);
        }
        j[key + '-' + name] = o;
        localStorage.setItem(this.WORKFLOW_ORIENTATION_KEY, JSON.stringify(j));
    }
}
