import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Rx';
import {Group} from '../../model/group.model';
import {Http} from '@angular/http';

/**
 * Service to access Group from API.
 * Only used by GroupStore
 */
@Injectable()
export class GroupService {


    constructor(private _http: Http) {
    }

    /**
     * Get all groups that the user can access.
     * @returns {Observable<Group[]>}
     */
    getGroups(): Observable<Group[]> {
        return this._http.get('/group').map(res => {
            return res.json();
        });
    }

    /**
     * Create a new group
     * @param group Group to create
     * @returns {Observable<boolean>}
     */
    addGroup(group: Group): Observable<boolean> {
        return this._http.post('/group', group).map(() => {
            return true;
        });
    }

    /**
     * Update the given group.
     * @param group Group updated
     * @returns {Observable<boolean>}
     */
    updateGroup(group: Group): Observable<boolean> {
        return this._http.put('/group/' + group.name, group).map(() => {
            return true;
        });
    }

    /**
     * Delete the given group
     * @param name Group name to delete
     * @returns {Observable<boolean>}
     */
    deleteGroup(name: string): Observable<boolean> {
        return this._http.delete('/group/' + name).map(() => {
            return true;
        });
    }
}
