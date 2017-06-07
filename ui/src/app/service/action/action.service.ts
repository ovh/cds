import {Injectable} from '@angular/core';
import {Http, Response} from '@angular/http';
import {Observable} from 'rxjs/Rx';
import {Action} from '../../model/action.model';

/**
 * Service to access Public Action
 */
@Injectable()
export class ActionService {

    constructor(private _http: Http) {
    }

    /**
     * Get all types of parameters
     * @returns {Observable<string[]>}
     */
    getActions(): Observable<Action[]> {
        return this._http.get('/action').map(res => res.json());
    }

    /**
     * Get a action by his name
     * @param name name of the action to get
     * @returns {Observable<Action>}
     */
    getAction(name: string): Observable<Action> {
        return this._http.get('/action/' + name).map(res => res.json());
    }

    /**
     * Update an action
     * @param action to update
     * @returns {Observable<Action>}
     */
    updateAction(action: Action): Observable<Action> {
        return this._http.put('/action/' + action.name, action).map(res => res.json());
    }

    /**
     * Delete a action from CDS
     * @param name Actionname of the action to delete
     * @returns {Observable<Response>}
     */
    deleteAction(name: string): Observable<Response> {
        return this._http.delete('/action/' + name);
    }
}
