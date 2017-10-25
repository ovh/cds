import {Injectable} from '@angular/core';
import {BehaviorSubject} from 'rxjs/BehaviorSubject';
import {Observable} from 'rxjs/Observable';
import {WorkflowRun} from '../../model/workflow.run.model';

@Injectable()
export class WorkflowCoreService {

    private _sideBarStatus: BehaviorSubject<boolean> = new BehaviorSubject(true);
    private _currentWorkflowRun: BehaviorSubject<WorkflowRun> = new BehaviorSubject(null);

    getSidebarStatus(): Observable<boolean> {
        return new Observable<boolean>(fn => this._sideBarStatus.subscribe(fn));
    }

    moveSideBar(o: boolean): void {
        this._sideBarStatus.next(o);
    }

    getCurrentWorkflowRun(): Observable<WorkflowRun> {
        return new Observable<WorkflowRun>(fn => this._currentWorkflowRun.subscribe(fn));
    }

    setCurrentWorkflowRun(wr: WorkflowRun): void {
        this._currentWorkflowRun.next(wr);
    }
}
