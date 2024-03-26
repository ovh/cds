import { Injectable } from '@angular/core';
import { BehaviorSubject, Observable } from 'rxjs';
import { HttpClient } from '@angular/common/http';
import { V2WorkflowRun } from '../../../../libs/workflow-graph/src/lib/v2.workflow.run.model';

export class SidebarEvent {
    nodeID: string
    nodeName: string
    nodeType: string
    action: string
    parentIDs: string[];

    constructor(nodeID: string, nodeName: string, type: string, action: string, parents: string[]) {
        this.action = action;
        this.nodeID = nodeID;
        this.nodeType = type;
        this.parentIDs = parents;
        this.nodeName = nodeName;
    }
}

@Injectable()
export class SidebarService {

    private _sidebarWorkspace: BehaviorSubject<SidebarEvent> = new BehaviorSubject(null);
    private _sidebarRun: BehaviorSubject<V2WorkflowRun> = new BehaviorSubject<V2WorkflowRun>(null);

    constructor(private _http: HttpClient) { }

    getWorkspaceObservable(): Observable<SidebarEvent> {
        return new Observable<SidebarEvent>(fn => this._sidebarWorkspace.subscribe(fn));
    }

    getRunObservable(): Observable<V2WorkflowRun> {
        return new Observable<V2WorkflowRun>(fn => this._sidebarRun.subscribe(fn));
    }

    sendEvent(event: SidebarEvent): void {
        if (!event.nodeID) {
            return;
        }
        this._sidebarWorkspace.next(event);
    }

    selectRun(r: V2WorkflowRun): void {
        this._sidebarRun.next(r)
    }

}
