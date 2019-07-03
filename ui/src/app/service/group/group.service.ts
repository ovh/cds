
import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';
import { Group } from '../../model/group.model';

@Injectable()
export class GroupService {

    constructor(
        private _http: HttpClient
    ) { }

    getByName(name: string): Observable<Group> {
        return this._http.get<Group>('/group/' + name);
    }

    getAll(withoutDefault?: boolean): Observable<Group[]> {
        let params = new HttpParams();
        if (withoutDefault === true) {
            params = params.append('withoutDefault', 'true');
        }
        return this._http.get<Group[]>('/group', { params: params });
    }

    create(group: Group): Observable<boolean> {
        return this._http.post('/group', group).pipe(map(() => {
            return true;
        }));
    }

    update(groupname: string, group: Group): Observable<boolean> {
        return this._http.put('/group/' + groupname, group).pipe(map(() => {
            return true;
        }));
    }

    delete(name: string): Observable<boolean> {
        return this._http.delete('/group/' + name).pipe(map(() => {
            return true;
        }));
    }

    addMember(name: string, username: string): Observable<boolean> {
        return this._http.post('/group/' + name + '/user', [username]).pipe(map(() => {
            return true;
        }));
    }

    removeMember(name: string, username: string): Observable<boolean> {
        return this._http.delete('/group/' + name + '/user/' + username).pipe(map(() => {
            return true;
        }));
    }

    addAdmin(name: string, username: string): Observable<boolean> {
        return this._http.post('/group/' + name + '/user/' + username + '/admin', null).pipe(map(() => {
            return true;
        }));
    }

    removeAdmin(name: string, username: string): Observable<boolean> {
        return this._http.delete('/group/' + name + '/user/' + username + '/admin').pipe(map(() => {
            return true;
        }));
    }

}
