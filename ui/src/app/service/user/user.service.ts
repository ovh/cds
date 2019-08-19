import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { AuthConsumer, AuthConsumerCreateResponse, AuthSession } from 'app/model/authentication.model';
import { Bookmark } from 'app/model/bookmark.model';
import { Group } from 'app/model/group.model';
import { AuthentifiedUser, UserContact } from 'app/model/user.model';
import { Observable } from 'rxjs';

@Injectable()
export class UserService {
    constructor(
        private _http: HttpClient
    ) { }

    getMe(): Observable<AuthentifiedUser> {
        return this._http.get<AuthentifiedUser>('/user/me').map(u => {
            return Object.assign(new AuthentifiedUser(), u);
        });
    }

    get(username: string): Observable<AuthentifiedUser> {
        return this._http.get<AuthentifiedUser>(`/user/${username}`).map(u => {
            return Object.assign(new AuthentifiedUser(), u);
        });
    }

    update(username: string, user: AuthentifiedUser): Observable<AuthentifiedUser> {
        return this._http.put<AuthentifiedUser>(`/user/${username}`, user).map(u => {
            return Object.assign(new AuthentifiedUser(), u);
        });
    }

    delete(username: string): Observable<Response> {
        return this._http.delete<Response>(`/user/${username}`);
    }

    getUsers(): Observable<Array<AuthentifiedUser>> {
        return this._http.get<Array<AuthentifiedUser>>('/user').map(us => {
            return us.map(u => Object.assign(new AuthentifiedUser(), u));
        });
    }

    getGroups(username: string): Observable<Array<Group>> {
        return this._http.get<Array<Group>>(`/user/${username}/groups`).map(gs => {
            return gs.map(g => Object.assign(new Group(), g));
        });
    }

    getContacts(username: string): Observable<Array<UserContact>> {
        return this._http.get<Array<UserContact>>(`/user/${username}/contacts`);
    }

    getConsumers(username: string): Observable<Array<AuthConsumer>> {
        return this._http.get<Array<AuthConsumer>>(`/user/${username}/auth/consumer`);
    }

    createConsumer(username: string, consumer: AuthConsumer): Observable<AuthConsumerCreateResponse> {
        return this._http.post<AuthConsumerCreateResponse>(`/user/${username}/auth/consumer`, consumer);
    }

    getSessions(username: string): Observable<Array<AuthSession>> {
        return this._http.get<Array<AuthSession>>(`/user/${username}/auth/session`);
    }

    deleteSession(username: string, sessionID: string): Observable<any> {
        return this._http.delete(`/user/${username}/auth/session/${sessionID}`);
    }

    getBookmarks(): Observable<Bookmark[]> {
        return this._http.get<Bookmark[]>('/bookmarks');
    }
}
