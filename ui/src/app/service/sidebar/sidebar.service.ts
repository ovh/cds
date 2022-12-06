import { Injectable } from '@angular/core';
import { BehaviorSubject, Observable } from 'rxjs';
import { HttpClient } from '@angular/common/http';
import { FlatNodeItem } from 'app/shared/tree/tree.component';

export class SidebarEvent {
    nodeID: string
    nodeName: string
    nodeType: string
    action: string
    parent: FlatNodeItem;

    constructor(nodeID: string, nodeName: string, type: string, action: string, parent: FlatNodeItem) {
        this.action = action;
        this.nodeID = nodeID;
        this.nodeType = type;
        this.parent = parent;
        this.nodeName = nodeName;
    }
}

@Injectable()
export class SidebarService {

    private _sidebar: BehaviorSubject<SidebarEvent> = new BehaviorSubject(null);

    constructor(private _http: HttpClient) { }

    getObservable(): Observable<SidebarEvent> {
        return new Observable<SidebarEvent>(fn => this._sidebar.subscribe(fn));
    }

    sendEvent(event: SidebarEvent): void {
        if (!event.nodeID) {
            return;
        }
        this._sidebar.next(event);
    }

}
