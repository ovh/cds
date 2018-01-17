import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Observable';
import {Group} from '../../model/group.model';
import {Token} from '../../model/token.model';
import {HttpClient} from '@angular/common/http';

/**
 * Service to access Group from API.
 * Only used by GroupStore
 */
@Injectable()
export class GroupService {


    constructor(private _http: HttpClient) {
    }

    /**
     * Get the list of groups
     * @returns {Observable<Group>}
     */
    getGroupByName(name: string): Observable<Group> {
        return this._http.get<Group>('/group/' + name);
    }

    /**
     * Get all groups that the user can access.
     * @returns {Observable<Group[]>}
     */
    getGroups(): Observable<Group[]> {
        return this._http.get<Group[]>('/group');
    }

    /**
     * Get all tokens linked to a specific group that the user can access.
     * @returns {Observable<Token[]>}
     */
    getTokens(groupName: string): Observable<Token[]> {
        return this._http.get<Token[]>(`/group/${groupName}/token`);
    }

    /**
     * Remove a token linked to a specific group that the user can access.
     * @returns {Observable<{}>}
     */
    removeToken(groupName: string, tokenId: number): Observable<{}> {
        return this._http.delete<{}>(`/group/${groupName}/token/${tokenId}`);
    }
    /**
     * Add a token linked to a specific group that the user can access.
     * @returns {Observable<Token>}
     */
    addToken(groupName: string, expiration: string, description: string): Observable<Token> {
        return this._http.post<Token>(`/group/${groupName}/token`, { expiration, description });
    }


    /**
     * Create a new group
     * @param group Group to create
     * @returns {Observable<boolean>}
     */
    createGroup(group: Group): Observable<boolean> {
        return this._http.post('/group', group).map(() => {
            return true;
        });
    }

    /**
     * Update the given group.
     * @param group Group updated
     * @returns {Observable<boolean>}
     */
    updateGroup(groupname: string, group: Group): Observable<boolean> {
        return this._http.put('/group/' + groupname, group).map(() => {
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

    /**
     * Add a user in a group
     * @param name Group name
     * @param user User to add into group
     * @returns {Observable<boolean>}
     */
    addUser(name: string, username: string): Observable<boolean> {
        return this._http.post('/group/' + name + '/user', [username]).map(() => {
            return true;
        });
    }

    /**
     * Remove user from group
     * @param name Group name
     * @param user User to remove from gropu
     * @returns {Observable<boolean>}
     */
    removeUser(name: string, username: string): Observable<boolean> {
        return this._http.delete('/group/' + name + '/user/' + username).map(() => {
            return true;
        });
    }

    /**
     * Add admin in a group
     * @param name Group name
     * @param user User to add
     * @returns {Observable<boolean>}
     */
    addUserAdmin(name: string, username: string): Observable<boolean> {
        return this._http.post('/group/' + name + '/user/' + username + '/admin', null).map(() => {
            return true;
        });
    }

    /**
     * Remove an admin from a group
     * @param name Group name
     * @param user user to add into group
     * @returns {Observable<boolean>}
     */
    removeUserAdmin(name: string, username: string): Observable<boolean> {
        return this._http.delete('/group/' + name + '/user/' + username + '/admin').map(() => {
            return true;
        });
    }

}
