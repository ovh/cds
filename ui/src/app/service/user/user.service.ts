import {Injectable} from '@angular/core';
import {Http, Response, RequestOptions, Headers} from '@angular/http';
import {User} from '../../model/user.model';
import {Observable} from 'rxjs/Rx';
import {AuthentificationStore} from '../auth/authentification.store';

@Injectable()
export class UserService {

    constructor(private _http: Http, private _authStore: AuthentificationStore) {
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
        return this._http.post('/login', user).map(res => {
            let u = res.json().user;
            let headers: Headers = res.headers;

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
        return this._http.post('/user/' + user.username + '/reset', request).map(res => {
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
        return this._http.post('/user/signup', request).map(res => {
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
        let options: RequestOptions = new RequestOptions();
        options.withCredentials = false;
        return this._http.get('/user/' + username + '/confirm/' + token, options).map(res => res.json());
    }

    /**
     * Get the list of all users.
     * @returns {Observable<User[]>}
     */
    getUsers(): Observable<User[]> {
        return this._http.get('/user').map(res => res.json());
    }

    /**
     * Get a user by his username
     * @param userName username of the user to get
     * @returns {Observable<User>}
     */
    getUser(userName: string): Observable<User> {
        return this._http.get('/user/' + userName).map(res => res.json());
    }

    /**
     * Update an user
     * @param user to update
     * @returns {Observable<User>}
     */
    update(user: User): Observable<User> {
        return this._http.put('/user/' + user.username, user).map(res => res.json());
    }

    /**
     * Delete a user from CDS
     * @param userName Username of the user to delete
     * @returns {Observable<Response>}
     */
    delete(userName: string): Observable<Response> {
        return this._http.delete('/user/' + userName);
    }
}
