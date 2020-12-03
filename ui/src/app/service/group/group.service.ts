import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Group, GroupMember } from 'app/model/group.model';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';

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
        return this._http.get<Group[]>('/group', { params }).pipe(map(gs => gs.map(g => Object.assign(new Group(), g))));
    }

    create(group: Group): Observable<Group> {
        return this._http.post<Group>('/group', group);
    }

    update(name: string, group: Group): Observable<Group> {
        return this._http.put<Group>(`/group/${name}`, group);
    }

    delete(name: string): Observable<boolean> {
        return this._http.delete(`/group/${name}`).pipe(map(() => true));
    }

    addMember(name: string, member: GroupMember): Observable<Group> {
        return this._http.post<Group>(`/group/${name}/user`, member);
    }

    updateMember(name: string, member: GroupMember): Observable<Group> {
        return this._http.put<Group>(`/group/${name}/user/${member.username}`, member);
    }

    removeMember(name: string, member: GroupMember): Observable<Group> {
        return this._http.delete<Group>(`/group/${name}/user/${member.username}`);
    }
}
