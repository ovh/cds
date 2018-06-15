import {Injectable} from '@angular/core';
import {OrderedMap} from 'immutable';
import {BehaviorSubject, Observable} from 'rxjs';
import {Action} from '../../model/action.model';
import {ActionService} from './action.service';

@Injectable()
export class ActionStore {

    // List of all public actions.
    private _actions: BehaviorSubject<OrderedMap<string, Action>> = new BehaviorSubject(OrderedMap<string, Action>());

    constructor(private _actionService: ActionService) {
    }

    /**
     * Get all actions
     * @returns {Observable<Application>}
     */
    getActions(): Observable<OrderedMap<string, Action>> {
        if (this._actions.getValue().size === 0) {
            this._actionService.getActions().subscribe(res => {
                let map = OrderedMap<string, Action>();
                if (res && res.length > 0) {
                    res.forEach(a => {
                        map = map.set(a.name, a);
                    });
                }
                this._actions.next(map);
            });
        }
        return new Observable<OrderedMap<string, Action>>(fn => this._actions.subscribe(fn));
    }
}
