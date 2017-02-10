import {Injectable} from '@angular/core';
import {Http} from '@angular/http';
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
}
