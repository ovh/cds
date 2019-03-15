import { Injectable } from '@angular/core';
import { OrderedMap } from 'immutable';
import { BehaviorSubject, Observable } from 'rxjs';
import { Action } from '../../model/action.model';
import { ActionService } from './action.service';

@Injectable()
export class ActionStore {
    projectActions: BehaviorSubject<OrderedMap<string, Action>> = new BehaviorSubject(OrderedMap<string, Action>());
    groupActions: BehaviorSubject<OrderedMap<string, Action>> = new BehaviorSubject(OrderedMap<string, Action>());
    projectKey: string;
    groupID: number;

    constructor(private _actionService: ActionService) { }

    getProjectActions(projectKey: string): Observable<OrderedMap<string, Action>> {
        if (this.projectActions.getValue().size === 0 || this.projectKey !== projectKey) {
            this.projectKey = projectKey;
            this.resyncForProject();
        }
        return new Observable<OrderedMap<string, Action>>(fn => this.projectActions.subscribe(fn));
    }

    getGroupActions(groupID: number): Observable<OrderedMap<string, Action>> {
        if (this.groupActions.getValue().size === 0 || this.groupID !== groupID) {
            this.groupID = groupID;
            this.resyncForGroup();
        }
        return new Observable<OrderedMap<string, Action>>(fn => this.groupActions.subscribe(fn));
    }

    resyncForProject(): void {
        this._actionService.getAllForProject(this.projectKey).subscribe(res => {
            let map = OrderedMap<string, Action>();
            if (res && res.length > 0) {
                res.forEach(a => {
                    map = map.set(a.name, a);
                });
            }
            this.projectActions.next(map);
        });
    }

    resyncForGroup(): void {
        this._actionService.getAllForGroup(this.groupID).subscribe(res => {
            let map = OrderedMap<string, Action>();
            if (res && res.length > 0) {
                res.forEach(a => {
                    map = map.set(a.name, a);
                });
            }
            this.groupActions.next(map);
        });
    }
}
