import {Injectable} from '@angular/core';
import {BehaviorSubject} from 'rxjs/BehaviorSubject';
import {Observable} from 'rxjs/Observable';

export class WorkflowSidebarMode {
    static RUNS = 'sidebar:runs';
    static RUN_NODE = 'sidebar:run:node';
    static RUN_HOOK = 'sidebar:run:hook';
    static EDIT_NODE = 'sidebar:edit:node';
    static EDIT_HOOK = 'sidebar:edit:hook';
    static EDIT_JOIN = 'sidebar:edit:join';
}

@Injectable()
export class WorkflowSidebarStore {

    private _sidebarMode: BehaviorSubject<string> = new BehaviorSubject(WorkflowSidebarMode.RUNS);

    constructor() {
    }

    sidebarMode(): Observable<string> {
        return new Observable<string>(fn => this._sidebarMode.subscribe(fn));
    }

    changeMode(m: string) {
        this._sidebarMode.next(m);
    }
}
