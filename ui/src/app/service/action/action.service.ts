import {Injectable} from '@angular/core';
import {Http, Response} from '@angular/http';
import {Observable} from 'rxjs/Rx';
import {Action, PipelineUsingAction} from '../../model/action.model';

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
     * Get pipelines using specified action
     * @param name name of the action to get
     * @returns {Observable<PipelineUsingAction>}
     */
    getPiplinesUsingAction(name: string): Observable<PipelineUsingAction[]> {
        return this._http.get('/action/' + name + '/using').map(res => res.json());
    }

    /**
     * Create an action
     * @param action to create
     * @returns {Observable<Action>}
     */
    createAction(action: Action): Observable<Action> {
        return this._http.post('/action/' + action.name, action).map(res => res.json());
    }

    /**
     * Update an action
     * @param action to update
     * @returns {Observable<Action>}
     */
    updateAction(name: string, action: Action): Observable<Action> {
        return this._http.put('/action/' + name, action).map(res => res.json());
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
