import { Injectable } from '@angular/core';
import { BehaviorSubject } from 'rxjs/BehaviorSubject';
import { Observable } from 'rxjs/Observable';

export class WorkflowSidebarMode {
    static EDIT = 'sidebar:edit';
    static EDIT_HOOK = 'sidebar:edit:hook';
    static RUNS = 'sidebar:runs';
    static RUN_NODE = 'sidebar:run:node';
    static RUN_HOOK = 'sidebar:run:hook';
}

@Injectable()
export class WorkflowSidebarStore {

    private _sidebarMode: BehaviorSubject<string> = new BehaviorSubject(null);

    constructor() {
    }

    sidebarMode(): Observable<string> {
        return new Observable<string>(fn => this._sidebarMode.subscribe(fn));
    }

    changeMode(m: string) {
        this._sidebarMode.next(m);
    }
}
