import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { AuthConsumer, AuthSession } from 'app/model/authentication.model';
import { Group } from 'app/model/group.model';
import { Observable } from 'rxjs';
import { Bookmark } from '../../model/bookmark.model';
import { User, UserContact } from '../../model/user.model';

@Injectable()
export class UserService {
    constructor(
        private _http: HttpClient
    ) { }

    getMe(): Observable<User> {
        return this._http.get<User>('/user/me');
    }

    getUsers(): Observable<User[]> {
        return this._http.get<User[]>('/user');
    }

    getGroups(username: string): Observable<Array<Group>> {
        return this._http.get<Array<Group>>('/user/' + username + '/groups');
    }

    getContacts(username: string): Observable<Array<UserContact>> {
        return this._http.get<Array<UserContact>>('/user/' + username + '/contacts');
    }

    getConsumers(username: string): Observable<Array<AuthConsumer>> {
        return this._http.get<Array<AuthConsumer>>(`/user/${username}/auth/consumer`);
    }

    getSessions(username: string): Observable<Array<AuthSession>> {
        return this._http.get<Array<AuthSession>>(`/user/${username}/auth/session`);
    }

    deleteSession(username: string, sessionID: string): Observable<any> {
        return this._http.delete(`/user/${username}/auth/session/${sessionID}`);
    }

    getUser(username: string): Observable<User> {
        return this._http.get<User>('/user/' + username);
    }

    updateUser(username: string, user: User): Observable<User> {
        return this._http.put<User>('/user/' + username, user);
    }

    deleteUser(username: string): Observable<Response> {
        return this._http.delete<Response>('/user/' + username);
    }

    getBookmarks(): Observable<Bookmark[]> {
        return this._http.get<Bookmark[]>('/bookmarks');
    }
}
