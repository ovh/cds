import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { Bookmark } from '../../model/bookmark.model';
import { Groups } from '../../model/group.model';
import { Token } from '../../model/token.model';
import { User } from '../../model/user.model';

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

    getTokens(): Observable<Token[]> {
        return this._http.get<Token[]>('/user/token');
    }

    getGroups(username: string): Observable<Groups> {
        return this._http.get<Groups>('/user/' + username + '/groups');
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
