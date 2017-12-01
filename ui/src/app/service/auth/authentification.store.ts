import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Observable';
import {BehaviorSubject} from 'rxjs/BehaviorSubject'
import {User} from '../../model/user.model';


@Injectable()
export class AuthentificationStore {

    // CDs user key on localstorage
    localStorageUserKey = 'CDS-USER';
    localStorageSessionKey = 'Session-Token';

    // Current connected user
    private _connectedUser: BehaviorSubject<User> = new BehaviorSubject<User>(null);

    /**
     * Constructor.
     */
    constructor() {
        // Init store at startup
        if (!localStorage.getItem(this.localStorageUserKey)) {
            return;
        }

        let user: User = JSON.parse(localStorage.getItem(this.localStorageUserKey));
        if (user) {
            this._connectedUser.next(user);
        }
    }

    /**
     * Get Observable to be aware of connection status.
     * @returns {Observable<User>}
     */
    getUserlst(): Observable<User> {
        return new Observable<User>(fn => {
            this._connectedUser.subscribe(fn);
        });
    }

    /**
     * Get the connected User
     * @returns {User}
     */
    getUser(): User {
        return this._connectedUser.getValue();
    }

    getSessionToken(): string {
        return localStorage.getItem(this.localStorageSessionKey);
    }

    /**
     * Check if user is connected
     * @returns {boolean}
     */
    isConnected(): boolean {
        // user is connected ?
        return this._connectedUser.getValue() != null;
    }

    /**
     * Check if user is admin
     * @returns {boolean}
     */
    isAdmin(): boolean {
        // user is connected ?
        if (!this.isConnected()) {
          return false;
        }
        // user is admin ?
        return this._connectedUser.getValue().admin;
    }

    /**
     * Remove user data from localstorage.
     */
    removeUser(): void {
        this._connectedUser.next(null);
        localStorage.setItem(this.localStorageUserKey, '');
        localStorage.setItem(this.localStorageSessionKey, '');
    }

    /**
     * Add user information in localstorage.
     * @param user User data to save in localstorage
     * @param session  Indicate if user.token is a session token or not
     */
    addUser(user: User, session: boolean): void {
        localStorage.setItem(this.localStorageUserKey, JSON.stringify(user));
        if (session) {
            localStorage.setItem(this.localStorageSessionKey, user.token);
        }
        this._connectedUser.next(user);
    }
}
