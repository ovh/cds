import {BehaviorSubject} from 'rxjs/BehaviorSubject';
import {Workflow, WorkflowTriggerConditionCache} from '../../model/workflow.model';
import {Injectable} from '@angular/core';
import {List, Map} from 'immutable';
import {Observable} from 'rxjs/Observable';
import {WorkflowService} from './workflow.service';
import {GroupPermission} from '../../model/group.model';
import {NavbarRecentData} from '../../model/navbar.model';
import 'rxjs/add/observable/of';
import {mergeMap, map} from 'rxjs/operators';

@Injectable()
export class WorkflowStore {

    static RECENT_WORKFLOWS_KEY = 'CDS-RECENT-WORKFLOWS';
    WORKFLOW_ORIENTATION_KEY = 'CDS-WORKFLOW-ORIENTATION';

    // List of all workflows.
    private _workflows: BehaviorSubject<Map<string, Workflow>> = new BehaviorSubject(Map<string, Workflow>());

    private _recentWorkflows: BehaviorSubject<List<NavbarRecentData>> = new BehaviorSubject(List<NavbarRecentData>());


    constructor(private _workflowService: WorkflowService) {
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
            this._workflows = new BehaviorSubject(Map<string, Workflow>());
            this._workflows.next(store);
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

    renameWorkflow(key: string, name: string, workflow: Workflow): Observable<Workflow> {
        return this._workflowService.updateWorkflow(key, name, workflow).map(w => {
            let workflowKey = key + '-' + workflow.name;
            let store = this._workflows.getValue();
            w.permission = workflow.permission;
            this._workflows.next(store.set(workflowKey, w));
            return w;
        });
    }

    /**
     * Update a workflow
     * @param key Project unique key
     * @param workflow workflow to update
     */
    updateWorkflow(key: string, workflow: Workflow): Observable<Workflow> {
        return this._workflowService.updateWorkflow(key, workflow.name, workflow).map(w => {
            let workflowKey = key + '-' + workflow.name;
            let store = this._workflows.getValue();
            this._workflows.next(store.set(workflowKey, w));
            return w;
        });
    }

    /**
     * Import a workflow
     * @param key Project unique key
     * @param workflow workflow to update
     */
    importWorkflow(key: string, workflowName: string, workflowCode: string): Observable<Workflow> {
        return this._workflowService.importWorkflow(key, workflowCode)
            .pipe(
                mergeMap(() => {
                  if (workflowName) {
                    return this._workflowService.getWorkflow(key, workflowName);
                  }
                  return Observable.of(null);
                }),
                map((wf) => {
                    if (wf) {
                      let workflowKey = key + '-' + wf.name;
                      let store = this._workflows.getValue();
                      this._workflows.next(store.set(workflowKey, wf));
                    }
                    return wf;
                })
            );
    }

    /**
     * Delete the given workflow
     * @param key Project unique key
     * @param workflow Workflow name
     */
    deleteWorkflow(key: string, workflow: Workflow): Observable<boolean> {
        return this._workflowService.deleteWorkflow(key, workflow).map(w => {
            let workflowKey = key + '-' + workflow.name;
            let store = this._workflows.getValue();
            this._workflows.next(store.delete(workflowKey));
            return w;
        });
    }

    getTriggerCondition(key: string, workflowName: string, nodeID: number): Observable<WorkflowTriggerConditionCache> {
        return this._workflowService.getTriggerCondition(key, workflowName, nodeID);
    }

    getTriggerJoinCondition(key: string, workflowName: string, joinID: number): any {
        return this._workflowService.getTriggerJoinCondition(key, workflowName, joinID);
    }

    getDirection(key: string, name: string) {
        let o = localStorage.getItem(this.WORKFLOW_ORIENTATION_KEY);
        if (o) {
            let j = JSON.parse(o);
            if (j[key + '-' + name]) {
                return j[key + '-' + name];
            }
        }
        return 'TB';
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

    addPermission(key: string, detailedWorkflow: Workflow, gp: GroupPermission) {
        return this._workflowService.addPermission(key, detailedWorkflow.name, gp).map(w => {
            let workflowKey = key + '-' + detailedWorkflow.name;
            let store = this._workflows.getValue();
            let workflowToUpdate = store.get(workflowKey);
            workflowToUpdate.groups = w.groups;
            workflowToUpdate.last_modified = w.last_modified;
            this._workflows.next(store.set(workflowKey, workflowToUpdate));
            return w;
        });
    }

    updatePermission(key: string, detailedWorkflow: Workflow, gp: GroupPermission) {
        return this._workflowService.updatePermission(key, detailedWorkflow.name, gp).map(w => {
            let workflowKey = key + '-' + detailedWorkflow.name;
            let store = this._workflows.getValue();
            let workflowToUpdate = store.get(workflowKey);
            workflowToUpdate.groups = w.groups;
            workflowToUpdate.last_modified = w.last_modified;
            this._workflows.next(store.set(workflowKey, workflowToUpdate));
            return w;
        });
    }

    deletePermission(key: string, detailedWorkflow: Workflow, gp: GroupPermission) {
        return this._workflowService.deletePermission(key, detailedWorkflow.name, gp).map(w => {
            let workflowKey = key + '-' + detailedWorkflow.name;
            let store = this._workflows.getValue();
            let workflowToUpdate = store.get(workflowKey);
            workflowToUpdate.groups = w.groups;
            workflowToUpdate.last_modified = w.last_modified;
            this._workflows.next(store.set(workflowKey, workflowToUpdate));
            return w;
        });
    }

    externalModification(wfKey: string) {
        let cache = this._workflows.getValue();
        let wfToUpdate = cache.get(wfKey);
        if (wfToUpdate) {
            wfToUpdate.externalChange = true;
            this._workflows.next(cache.set(wfKey, wfToUpdate));
        }
    }

    removeFromStore(wfKey: string) {
        let cache = this._workflows.getValue();
        this._workflows.next(cache.delete(wfKey));
    }
}
