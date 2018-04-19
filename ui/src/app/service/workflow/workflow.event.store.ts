import {Injectable} from '@angular/core';
import {BehaviorSubject} from 'rxjs/BehaviorSubject';
import {WorkflowNodeRun, WorkflowRun} from '../../model/workflow.run.model';
import {Map} from 'immutable';
import {Observable} from 'rxjs/Observable';
import {WorkflowNode, WorkflowNodeHook, WorkflowNodeJoin} from '../../model/workflow.model';
import {WorkflowSidebarMode, WorkflowSidebarStore} from './workflow.sidebar.store';
import {WorkflowService} from './workflow.service';

@Injectable()
export class WorkflowEventStore {

    private _currentWorkflowRuns: BehaviorSubject<Map<number, WorkflowRun>> = new BehaviorSubject(Map<number, WorkflowRun>());
    private _currentWorkflowRun: BehaviorSubject<WorkflowRun> = new BehaviorSubject(null);
    private _currentWorkflowNodeRun: BehaviorSubject<WorkflowNodeRun> = new BehaviorSubject(null);
    private _nodeRunEvents: BehaviorSubject<WorkflowNodeRun> = new BehaviorSubject(null);

    private _selectedNode: BehaviorSubject<WorkflowNode> = new BehaviorSubject<WorkflowNode>(null);
    private _selectedJoin: BehaviorSubject<WorkflowNodeJoin> = new BehaviorSubject<WorkflowNodeJoin>(null);
    private _selectedHook: BehaviorSubject<WorkflowNodeHook> = new BehaviorSubject<WorkflowNodeHook>(null);


    constructor(private _sidebarStore: WorkflowSidebarStore, private _workflowService: WorkflowService) {
    }

    broadcastWorkflowRun(key: string, name: string, wr: WorkflowRun): void {
        let store = this._currentWorkflowRuns.getValue();
        let w = store.get(wr.id);
        // Update workflow runs list
        if (!w || (w && (new Date(wr.last_modified).getTime() > (new Date(w.last_modified)).getTime())) ) {
            this._currentWorkflowRuns.next(store.set(wr.id, wr));
        }

        let sRun = this._currentWorkflowRun.getValue();
        if (sRun && sRun.id === wr.id && new Date(wr.last_modified).getTime() > new Date(sRun.last_modified).getTime()) {
            if (!wr.workflow) {
                this._workflowService.getWorkflow(key, name).subscribe(work => {
                    wr.workflow = work;
                    this._currentWorkflowRun.next(wr);
                });
            } else {
                this._currentWorkflowRun.next(wr);
            }
        }
    }

    pushWorkflowRuns(wrs: Array<WorkflowRun>): void {
        if (wrs) {
            let store = Map<number, WorkflowRun>();
            wrs.forEach(wr => {
                store = store.set(wr.id, wr);
            });
            this._currentWorkflowRuns.next(store);
        }
    }

    workflowRuns(): Observable<Map<number, WorkflowRun>> {
        return new Observable<Map<number, WorkflowRun>>(fn => this._currentWorkflowRuns.subscribe(fn));
    }

    isRunSelected(): boolean {
        return this._currentWorkflowRun.getValue() != null;
    }

    setSelectedNode(n: WorkflowNode) {
        if (n) {
            this._sidebarStore.changeMode(WorkflowSidebarMode.EDIT_NODE);
        }
        this._selectedNode.next(n);
        this._selectedJoin.next(null);
        this._selectedHook.next(null);
    }

    selectedNode(): Observable<WorkflowNode> {
        return new Observable<WorkflowNode>(fn => this._selectedNode.subscribe(fn));
    }

    setSelectedJoin(n: WorkflowNodeJoin) {
        if (n) {
            this._sidebarStore.changeMode(WorkflowSidebarMode.EDIT_JOIN);
        }
        this._selectedNode.next(null);
        this._selectedJoin.next(n);
        this._selectedHook.next(null);
    }

    selectedJoin(): Observable<WorkflowNodeJoin> {
        return new Observable<WorkflowNodeJoin>(fn => this._selectedJoin.subscribe(fn));
    }

    setSelectedHook(h: WorkflowNodeHook) {
        if (h) {
            this._sidebarStore.changeMode(WorkflowSidebarMode.EDIT_HOOK);
        }
        this._selectedNode.next(null);
        this._selectedJoin.next(null);
        this._selectedHook.next(h);
    }

    selectedHook(): Observable<WorkflowNodeHook> {
        return new Observable<WorkflowNodeHook>(fn => this._selectedHook.subscribe(fn));
    }

    unselectAll(): void {
        this._selectedNode.next(null);
        this._currentWorkflowRun.next(null);
        this._selectedHook.next(null);
        this._selectedJoin.next(null);
        this._sidebarStore.changeMode(WorkflowSidebarMode.RUNS);
    }

    setSelectedRun(wr: WorkflowRun) {
        if (wr) {
            this._sidebarStore.changeMode(WorkflowSidebarMode.RUNS);
        }
        this._currentWorkflowRun.next(wr);
    }

    selectedRun(): Observable<WorkflowRun> {
        return new Observable<WorkflowRun>(fn => this._currentWorkflowRun.subscribe(fn));
    }

    setSelectedNodeRun(wnr: WorkflowNodeRun) {
        let current = this._currentWorkflowNodeRun.getValue();
        if (wnr) {
            this._sidebarStore.changeMode(WorkflowSidebarMode.RUN_NODE);
            if (current && current.id === wnr.id) {
                // update value
                current.status = wnr.status;
            }
        }
        current = wnr;
        this._currentWorkflowNodeRun.next(current);
    }

    selectedNodeRun(): Observable<WorkflowNodeRun> {
        return new Observable<WorkflowNodeRun>(fn => this._currentWorkflowNodeRun.subscribe(fn));
    }

    broadcastNodeRunEvents(wnr: WorkflowNodeRun) {
        this._nodeRunEvents.next(wnr);

        let sNR = this._currentWorkflowNodeRun.getValue();
        if (sNR && sNR.id === wnr.id) {
            this._currentWorkflowNodeRun.next(wnr);
        }
    }

    nodeRunEvents(): Observable<WorkflowNodeRun> {
        return new Observable<WorkflowNodeRun>(fn => this._nodeRunEvents.subscribe(fn));
    }
}
