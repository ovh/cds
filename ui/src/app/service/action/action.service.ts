import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {Action, PipelineUsingAction} from '../../model/action.model';

/**
 * Service to access Public Action
 */
@Injectable()
export class ActionService {

    constructor(private _http: HttpClient) {
    }

    /**
     * Get all types of parameters
     * @returns {Observable<string[]>}
     */
    getActions(): Observable<Action[]> {
        return this._http.get<Action[]>('/action');
    }

    /**
     * Get a action by his name
     * @param name name of the action to get
     * @returns {Observable<Action>}
     */
    getAction(name: string): Observable<Action> {
        return this._http.get<Action>('/action/' + name);
    }

    /**
     * Get pipelines using specified action
     * @param name name of the action to get
     * @returns {Observable<PipelineUsingAction>}
     */
    getPiplinesUsingAction(name: string): Observable<PipelineUsingAction[]> {
        return this._http.get<PipelineUsingAction[]>('/action/' + name + '/using');
    }

    /**
     * Create an action
     * @param action to create
     * @returns {Observable<Action>}
     */
    createAction(action: Action): Observable<Action> {
        return this._http.post<Action>('/action/' + action.name, action);
    }

    /**
     * Update an action
     * @param action to update
     * @returns {Observable<Action>}
     */
    updateAction(name: string, action: Action): Observable<Action> {
        return this._http.put<Action>('/action/' + name, action);
    }

    /**
     * Delete a action from CDS
     * @param name Actionname of the action to delete
     * @returns {Observable<Response>}
     */
    deleteAction(name: string): Observable<Response> {
        return this._http.delete<Response>('/action/' + name);
    }
}
