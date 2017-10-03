import {Injectable} from '@angular/core';
import {BehaviorSubject} from 'rxjs/BehaviorSubject';
import {Observable} from 'rxjs/Observable';

@Injectable()
export class WorkflowCoreService {

    private _sideBarStatus: BehaviorSubject<boolean> = new BehaviorSubject(true);

    get(): Observable<boolean> {
        return new Observable<boolean>(fn => this._sideBarStatus.subscribe(fn));
    }

    moveSideBar(o: boolean): void {
        this._sideBarStatus.next(o);
    }
}
