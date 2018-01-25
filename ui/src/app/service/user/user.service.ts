import {Injectable} from '@angular/core';
import {User} from '../../model/user.model';
import {Token} from '../../model/token.model';
import {Groups} from '../../model/group.model';
import {Observable} from 'rxjs/Observable';
import {AuthentificationStore} from '../auth/authentification.store';
import {HttpClient, HttpHeaders} from '@angular/common/http';

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
        return this._http.post<any>('/login', user, {observe: 'response'}).map(res => {
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
        });
    }

    resetPassword(user: User, href: string) {
        let request = {
            user: user,
            callback: href + 'account/verify/%s/%s'
        };
        return this._http.post('/user/' + user.username + '/reset', request).map(() => {
            return true;
        });
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
        return this._http.post('/user/signup', request).map(() => {
            return true;
        });
    }

    /**
     * Verify user token to activate his account.
     * @param username Username to activate
     * @param token activation token
     * @returns {Observable<Response>}
     */
    verify(username: string, token: string): Observable<Response> {
        return this._http.get<Response>('/user/' + username + '/confirm/' + token);
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
}
