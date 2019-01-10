
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';
import { Bookmark } from '../../model/bookmark.model';
import { Groups } from '../../model/group.model';
import { Token } from '../../model/token.model';
import { User } from '../../model/user.model';
import { AuthentificationStore } from '../auth/authentification.store';

@Injectable()
export class UserService {

    constructor(private _http: HttpClient, private _authStore: AuthentificationStore) {
    }

    /**
     * Disconnect the user
     */
    disconnect() {
        // disconnect the user
        this._authStore.removeUser();
    }

    /**
     * LogIn user to API
     * @param user User to login
     * @returns {Observable<User>}
     */
    login(user: User): Observable<User> {
        return this._http.post<any>('/login', user, {observe: 'response'}).pipe(map(res => {
            let u = res.body.user;
            let headers: HttpHeaders = res.headers;

            let sessionToken: string = null;
            if (headers) {
                sessionToken = headers.get(this._authStore.localStorageSessionKey);
            }

            if (sessionToken) {
                u.token = sessionToken;
                this._authStore.addUser(u, true);
            } else {
                u.token = btoa(u.username + ':' + user.password);
                this._authStore.addUser(u, false);
            }
            return u;
        }));
    }

    resetPassword(user: User, href: string) {
        let request = {
            user: user,
            callback: href + 'account/verify/%s/%s'
        };
        return this._http.post('/user/' + user.username + '/reset', request).pipe(map(() => {
            return true;
        }));
    }

    /**
     * Create new CDS User
     * @param user CDS user to add
     * @returns {Observable<Boolean>}
     */
    signup(user: User, href: string): Observable<Boolean> {
        let request = {
            user: user,
            callback: href + 'account/verify/%s/%s'
        };
        return this._http.post('/user/signup', request).pipe(map(() => {
            return true;
        }));
    }

    /**
     * Verify user token to activate his account.
     * @param username Username to activate
     * @param token activation token
     * @returns {Observable<any>}
     */
    verify(username: string, token: string): Observable<any> {
        return this._http.get<any>('/user/' + username + '/confirm/' + token);
    }

    /**
     * Get the list of all users.
     * @returns {Observable<User[]>}
     */
    getUsers(): Observable<User[]> {
        return this._http.get<User[]>('/user');
    }

    /**
     * Get the list of all tokens for a user.
     * @returns {Observable<Token[]>}
     */
    getTokens(): Observable<Token[]> {
        return this._http.get<Token[]>('/user/token');
    }

    /**
     * Get user groups.
     * @returns {Observable<User[]>}
     */
    getGroups(username: string): Observable<Groups> {
        return this._http.get<Groups>('/user/' + username + '/groups');
    }

    /**
     * Get a user by his username
     * @param username username of the user to get
     * @returns {Observable<User>}
     */
    getUser(username: string): Observable<User> {
        return this._http.get<User>('/user/' + username);
    }

    /**
     * Update an user
     * @param username to update
     * @param user new values
     * @returns {Observable<User>}
     */
    updateUser(username: string, user: User): Observable<User> {
        return this._http.put<User>('/user/' + username, user);
    }

    /**
     * Delete a user from CDS
     * @param username Username of the user to delete
     * @returns {Observable<Response>}
     */
    deleteUser(username: string): Observable<Response> {
        return this._http.delete<Response>('/user/' + username);
    }

    /**
     * Get bookmarks for current user
     * @returns {Observable<Bookmark>}
     */
    getBookmarks(): Observable<Bookmark[]> {
        return this._http.get<Bookmark[]>('/bookmarks');
    }
}
